# PTMGR — Push Token Manager

## What It Does

PTMGR manages Firebase Cloud Messaging (FCM) tokens for mobile push notifications. When
a mobile app registers for push notifications, the FCM token is stored in PTMGR. UNH
then queries PTMGR to find the right tokens when sending push notifications.

Use PTMGR when your platform includes mobile apps that need push notifications.

## Architecture

```
┌──────────────┐   REST    ┌───────────┐
│  Mobile App  │─────────►│ PTMGR-App │  (register/refresh FCM token)
└──────────────┘          └─────┬─────┘
                                │
                           ┌────▼──────┐
                           │ PTMGR-DB  │
                           │ (Postgres)│
                           └───────────┘
                                │
┌──────────────┐   query   ┌────▼──────┐
│   UNH-App   │─────────►│ PTMGR-App │  (lookup tokens by user ID)
│ (push send)  │          │           │
└──────────────┘          └───────────┘
```

PTMGR is API-only — no web frontend.

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **ptmgr-db** | `postgres:17.0` | FCM token storage |
| **ptmgr-app** | `ghcr.io/mssfoobar/ptmgr/ptmgr-app` | Token CRUD API |

## Key Concepts

### FCM tokens

Each mobile device gets a unique FCM token from Firebase when the app registers for
push notifications. PTMGR stores these tokens mapped to users:

- **token_id** — The FCM token string
- **user_id** — The AOH user (from Keycloak)
- **device_id** — Unique device identifier

One user can have multiple tokens (multiple devices). One device has one token at a time.

### Token lifecycle

1. Mobile app obtains FCM token from Firebase
2. App registers token with PTMGR (`PUT /token` with user_id + device_id + token_id)
3. When token refreshes (Firebase rotates tokens), app calls the same PUT endpoint
   — it creates a new token or refreshes if one already exists (upsert)

### Push notification flow

1. Backend sends notification via UNH with push channel
2. UNH queries PTMGR for the recipient's FCM tokens
3. UNH sends push notification to Firebase using the tokens
4. Firebase delivers to the mobile device

## API Endpoints (PTMGR-App)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `PUT` | `/token` | Create or refresh FCM token (upsert by user_id + device_id) |

## Data Model

```
token request (PUT /token body):
  token_id: string (the FCM token)
  user_id: string (Keycloak user ID)
  device_id: string (unique device identifier)
```

## Auth

PTMGR uses a **confidential** Keycloak client (`ptmgr`) with a service account.
It integrates with both Keycloak (token validation) and AAS (authorization checks).

## Access

| URL | What |
|-----|------|
| `http://ptmgr.${DEV_DOMAIN}` | PTMGR App API |
| `http://ptmgr.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |

## Dependencies

- **iams** (Keycloak + AAS for auth)

## Related Services

- **UNH** — Uses PTMGR to look up FCM tokens when sending push notifications
