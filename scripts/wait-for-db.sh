#!/bin/sh
set -e

host="$1"
shift
cmd="$@"

until wget -q http://$host:8080/health?ready=1 -O /dev/null; do
  >&2 echo "db is unavailable - sleeping"
  sleep 1
done

>&2 echo "db is up - executing command"
exec $cmd