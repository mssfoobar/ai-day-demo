# Forward proxy (Squid)

A Squid forward proxy listening on port 3128. Clients may reach two
destination addresses only; everything else is denied.

## Configure

Edit `squid.conf` and replace the two placeholders:

| Placeholder  | Meaning                     |
| ------------ | --------------------------- |
| `DEST_IP_1`  | First allowed destination   |
| `DEST_IP_2`  | Second allowed destination  |

Each accepts a bare IP (`203.0.113.10`) or a CIDR range
(`203.0.113.0/24`).

## Run

```sh
docker compose up -d
```

## Verify

```sh
docker compose exec squid squid -k parse
curl -x http://localhost:3128 http://DEST_IP_1/
```

The second command should return the origin response, and any other host
should return `403 Forbidden`.

## Notes

- `localnet` covers the RFC 1918 and link-local ranges. Narrow it to your
  client subnet before exposing the port beyond the host.
- `forwarded_for delete` and `via off` strip the client address and the
  proxy signature from forwarded requests.
- Caching is disabled by `cache deny all`.
