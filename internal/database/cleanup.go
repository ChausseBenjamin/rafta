package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/ChausseBenjamin/rafta/internal/logging"
	"github.com/google/uuid"
)

// revocationCacheGrace sets how long a revoked token should remain in the database
// even though it is expired.
const revocationCacheGrace = 2 * time.Minute

// CleanupExpiredToken cleans up a revoked token if it is expired.
// It sets a timer to clean up the token after its expiry plus a grace period.
// The idea here is that even a valid (non-revoked) token would get denied if it's
// expired so it's useless to keep it in the database.
func (q *Queries) CleanupExpiredToken(ctx context.Context, tokenID uuid.UUID) error {
	token, err := q.GetRevokedToken(ctx, tokenID)
	if err != nil {
		return err
	}

	log := slog.With(
		"token_id", token.TokenID,
		"expiry", token.Expiry,
	)
	log.DebugContext(ctx, "Token scheduled for cleanup after expiry")

	// Just cleanup now if the token is already expired (good for after a reboot)
	if token.Expiry.Add(revocationCacheGrace).Before(time.Now()) {
		err := q.CleanRevokedToken(ctx, tokenID)
		if err != nil {
			log.ErrorContext(ctx,
				"An error occurred cleaning up an expired token",
				logging.ErrKey, err,
			)
			return err
		} else {
			log.InfoContext(ctx, "cleaned up already expired token")
		}
	}

	timer := time.NewTimer(time.Until(token.Expiry) + revocationCacheGrace)
	go func() {
		defer timer.Stop()
		<-timer.C
		// cleanup SQL command isn't context dependent (shouldn't fail if context
		// is expired). Context is only used to know where a deletetion request
		// originated from in logs.
		err := q.CleanRevokedToken(context.Background(), tokenID)
		if err != nil {
			log.ErrorContext(ctx,
				"An error occurred cleaning up an expired token",
				logging.ErrKey, err,
			)
			return
		}
		log.InfoContext(ctx, "Revoked token now expired: removed it from database")
	}()
	return nil
}

// TokenCleanupProcess retrieves all revoked tokens and runs
// CleanupExpiredToken for each one. This is meant to be run at startup to
// ensure the database doesn't get clogged up with revoked tokens that are
// expired anyways.
func (q *Queries) tokenCleanupProcess(ctx context.Context) error {
	tokens, err := q.GetAllRevokedTokens(ctx)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		err := q.CleanupExpiredToken(ctx, token.TokenID)
		if err != nil {
			return err
		}
	}

	return nil
}
