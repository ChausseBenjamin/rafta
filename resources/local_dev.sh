#!/bin/sh

env_vars=$(cat << EOF
  LOG_LEVEL=debug
  LOG_FORMAT=plain
  LOG_OUTPUT=stdout
  LISTEN_PORT=1157
  DATABASE_PATH=runtime/store.db
  GRACEFUL_TIMEOUT=1ms
EOF
)

case "$1" in
  --get-config)
    echo "$env_vars"
    ;;
  *)
    clear && env $env_vars go run . $@
    ;;
esac
