#!/bin/sh

localsecrets=".secrets"

mkdir -p "$localsecrets" || exit 1
[ -f "$localsecrets/.gitignore" ] || echo '*' > "$localsecrets/.gitignore"

env_vars=$(cat << EOF
  LOG_LEVEL=debug
  LOG_FORMAT=plain
  LOG_OUTPUT=stdout
  LISTEN_PORT=1157
  DATABASE_PATH=runtime/store.db
  GRACEFUL_TIMEOUT=200ms
  SECRETS_PATH=$localsecrets
EOF
)

case "$1" in
  --print-config)
    echo "$env_vars"
    ;;
  *)
    clear && env $env_vars go run . $@
    ;;
esac
