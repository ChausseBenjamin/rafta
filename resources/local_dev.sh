#!/bin/sh

env_vars=$(cat << EOF
  LOG_LEVEL=debug
  LOG_FORMAT=plain
  LOG_OUTPUT=stdout
  LISTEN_PORT=1234
  DATABASE_PATH=runtime/store.db
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
