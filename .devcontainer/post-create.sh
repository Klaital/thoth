#!/usr/bin/env bash
# Runs once when the devcontainer is created.
set -euo pipefail

# Default `mysql` / `mysqldump` to the compose database.
umask 077
cat > ~/.my.cnf <<CNF
[client]
host=db
user=${MYSQL_USER}
password=${MYSQL_PASSWORD}

[mysql]
database=${MYSQL_DATABASE}
CNF

npm ci --prefix frontend
make proto sqlc
