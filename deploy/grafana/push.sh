#!/usr/bin/env bash
# Upserts every dashboard JSON in this directory into the "Thoth" folder of a
# Grafana instance. The JSON here is the source of truth: dashboards are
# overwritten, so make changes in git (edit in the UI, then Export > JSON and
# commit it here).
#
#   GRAFANA_URL=https://<stack>.grafana.net GRAFANA_TOKEN=glsa_... deploy/grafana/push.sh
#   GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin deploy/grafana/push.sh   # local otel-lgtm
#
# GRAFANA_TOKEN is a service account token (Editor role is enough).
set -euo pipefail

: "${GRAFANA_URL:?GRAFANA_URL is required}"
dir="$(cd "$(dirname "$0")" && pwd)"
folder_uid=thoth
message="${GRAFANA_MESSAGE:-pushed by deploy/grafana/push.sh}"

if [[ -n "${GRAFANA_TOKEN:-}" ]]; then
    auth=(-H "Authorization: Bearer ${GRAFANA_TOKEN}")
elif [[ -n "${GRAFANA_AUTH:-}" ]]; then
    auth=(-u "${GRAFANA_AUTH}")
else
    echo "set GRAFANA_TOKEN (service account) or GRAFANA_AUTH (user:password)" >&2
    exit 1
fi

api() { # api METHOD PATH [JSON]  -> prints body, fails on HTTP >= 400
    local method=$1 path=$2 body=${3:-}
    curl -sS --fail-with-body -X "$method" "${auth[@]}" \
        -H 'Content-Type: application/json' \
        ${body:+--data-binary "$body"} "${GRAFANA_URL%/}$path"
}

if ! api GET "/api/folders/${folder_uid}" >/dev/null 2>&1; then
    api POST /api/folders "{\"uid\":\"${folder_uid}\",\"title\":\"Thoth\"}" >/dev/null
    echo "created folder Thoth"
fi

for f in "$dir"/*.json; do
    payload=$(jq --arg folder "$folder_uid" --arg msg "$message" \
        '{dashboard: (del(.id)), folderUid: $folder, overwrite: true, message: $msg}' "$f")
    result=$(api POST /api/dashboards/db "$payload")
    echo "$(basename "$f"): $(jq -r '"\(.status) v\(.version) \(.url)"' <<<"$result")"
done
