#!/bin/sh

localsecrets=".secrets"
localruntime=".runtime"

binpath="build/rafta"

mkdir -p "$localsecrets" || exit 1
[ -f "$localsecrets/.gitignore" ] || echo '*' > "$localsecrets/.gitignore"

mkdir -p "$localruntime" || exit 1
[ -f "$localruntime/.gitignore" ] || echo '*' > "$localruntime/.gitignore"

# Values here are not production ready, they are meant to ease development
env_vars=$(cat << EOF
  LOG_LEVEL=debug
  LOG_FORMAT=plain
  LOG_OUTPUT=stdout
  LISTEN_PORT=1157
  DATABASE_PATH=$localruntime/store.db
  GRACEFUL_TIMEOUT=200ms
  SECRETS_PATH=$localsecrets
  JWT_ACCESS_TTL=720h
  JWT_REFRESH_TTL=1m
  CGO_ENABLED=1
EOF
)


case "$1" in
  --print-config)
    echo "$env_vars"
    ;;
  --bin)
    clear && shift && env $env_vars "$binpath" $@
    ;;
  *)
    clear && env $env_vars go run . $@
    ;;
esac
