#!/usr/bin/env bash
# Seed geo-entities into gis-service so markers appear on the map without
# first building a backend publisher (the data path is backend -> RTUS ->
# map; gis-service forwards upserts to RTUS — see references/entities.md,
# "the data-source rule").
#
# Idempotent: PUT /geoentity upserts by entity_id.
#
# Usage:
#   ./seed-geoentities.sh                  # seed two sample "demo" markers
#   ./seed-geoentities.sh entities.json    # seed your own (JSON array of
#                                          # {entity_id, entity_type, geojson})
#
# Config (env): GIS_URL, KC_URL, DEV_USER, DEV_PASSWORD — defaults match the
# aoh-compose dev stack. If you changed TRAEFIK_HTTP_PORT, set GIS_URL and
# KC_URL accordingly (the defaults assume port 80).
set -euo pipefail

KC_URL="${KC_URL:-http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/protocol/openid-connect/token}"
GIS_URL="${GIS_URL:-http://gis.127.0.0.1.nip.io}"
DEV_USER="${DEV_USER:-admin}"
DEV_PASSWORD="${DEV_PASSWORD:-P@ssw0rd}"

# scope=openid is required: AOH services validate tokens via Keycloak's
# userinfo endpoint, which rejects tokens missing it (opaque 403 otherwise).
# See aoh-knowledge/references/services/iams.md.
TOKEN=$(curl -fsS -X POST "$KC_URL" \
	-d 'grant_type=password' -d 'client_id=web' -d 'scope=openid' \
	--data-urlencode "username=$DEV_USER" --data-urlencode "password=$DEV_PASSWORD" |
	python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")

put_entity() { # args: one geo-entity JSON object on stdin
	curl -fsS -o /dev/null -w "PUT %{http_code}\n" -X PUT "$GIS_URL/geoentity" \
		-H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
		-d @-
}

if [[ $# -ge 1 ]]; then
	# User-supplied JSON array — PUT each element.
	python3 -c "import json,sys; [print(json.dumps(e)) for e in json.load(open(sys.argv[1]))]" "$1" |
		while IFS= read -r entity; do put_entity <<<"$entity"; done
else
	# Two sample markers (kind=demo), Singapore area. Render them with:
	#   <MapEntityLayerProvider layer_name="Demo">
	#     <MapEntityProvider kind="demo" ... />
	#   </MapEntityLayerProvider>
	for i in 1 2; do
		lon=$([[ $i = 1 ]] && echo 103.8198 || echo 103.8607)
		lat=$([[ $i = 1 ]] && echo 1.3521 || echo 1.2834)
		put_entity <<JSON
{
  "entity_id": "DEMO-${i}",
  "entity_type": "static",
  "geojson": {
    "type": "Feature",
    "geometry": { "type": "Point", "coordinates": [${lon}, ${lat}] },
    "properties": { "kind": "demo", "name": "Demo marker ${i}" }
  }
}
JSON
	done
fi

echo "Seed complete."
