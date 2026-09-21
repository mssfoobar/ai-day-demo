## ADDED Requirements

### Requirement: Operators sign in through IAMS

The console SHALL authenticate operators through `iams-keycloak` using OpenID Connect
Authorization Code flow with PKCE against the bundled `aoh` realm. It SHALL reuse the
realm's existing public `web` client. This change SHALL NOT create a realm, SHALL NOT add a
realm role, SHALL NOT add a claim mapper, and SHALL NOT register a second public client.

#### Scenario: An unauthenticated visitor is sent to sign in
- **WHEN** a visitor with no session opens any console route
- **THEN** they are redirected to the `aoh` realm's authorization endpoint
- **AND** the redirect carries a `code_challenge` and `code_challenge_method=S256`

#### Scenario: A successful sign-in lands on the console
- **WHEN** a seeded realm user completes the Keycloak login form
- **THEN** they are returned to the console at `/aoh/dispatch/units` with a session established
- **AND** the roster renders

#### Scenario: Bare root leads to sign-in, not to the console
- **WHEN** a visitor with no session opens `/`
- **THEN** they end at the sign-in flow and not at an unauthenticated console page

#### Scenario: The console adds no public client
- **WHEN** `compose/iams/keycloak/realm-import.json` is compared against the platform-shipped file
- **THEN** it declares no added public client, no added realm role, and no added claim mapper
- **AND** the only client this change adds is the confidential one specified below

### Requirement: The projection worker authenticates as a service account

The outbox worker SHALL NOT reuse an operator's token, because it calls `gis-service` after
the originating request has returned and a retry may happen long after that token expires.
It SHALL obtain its own token through the
client-credentials grant using a single confidential client, `dispatch-svc`, declared in
`compose/iams/keycloak/realm-import.json` with `serviceAccountsEnabled` and the `openid`
scope. That client's service-account user SHALL be a member of the `development` tenant so
`gis-service` accepts its writes. No operator token, refresh token, or credential SHALL be
persisted in the outbox.

#### Scenario: The worker obtains its own token
- **WHEN** the worker needs to deliver a projection
- **THEN** it requests a token with the client-credentials grant for the `dispatch-svc` client
- **AND** `gis-service` accepts that token

#### Scenario: The outbox holds no credentials
- **WHEN** an outbox row is inspected
- **THEN** it carries the target unit, the intent and the payload
- **AND** it carries no access token, refresh token, or client secret

#### Scenario: The service account belongs to the development tenant
- **WHEN** a client-credentials token for `dispatch-svc` is obtained and decoded
- **THEN** its `active_tenant.tenant_id` is the `development` tenant's id

#### Scenario: The client is present after a fresh import
- **WHEN** the stack is brought up from empty volumes
- **THEN** the `dispatch-svc` client exists in the `aoh` realm without any manual step

### Requirement: Session tokens are held server-side in SDS

The console SHALL store the access and refresh tokens in `sds-server`. The browser SHALL
receive only an opaque session-id cookie. No access token, refresh token, or ID token
SHALL be readable by the browser or serialised into any page payload.

#### Scenario: Only a session id reaches the browser
- **WHEN** an operator has signed in and the browser's cookies are inspected
- **THEN** a `web_auth_session_id` cookie is present
- **AND** no cookie, `localStorage` entry, or server-rendered payload contains a JWT

#### Scenario: The cookie prefix matches the platform default
- **WHEN** the app's environment is inspected
- **THEN** `PUBLIC_COOKIE_PREFIX` is `web`, so the cookie name matches the `rtus.session-id.cookienames` value seeded on `rtus-seh`

#### Scenario: The server resolves the session to a token
- **WHEN** a request carrying a valid session-id cookie reaches a server route
- **THEN** the server obtains the access token from SDS for that session
- **AND** the request to `dispatch-svc` carries it as an `Authorization: Bearer` header

### Requirement: Signing out ends the session

The console SHALL offer a sign-out that destroys the SDS store for the session and ends
the Keycloak session. A session id captured before sign-out SHALL NOT be usable
afterwards.

#### Scenario: Sign-out returns the operator to sign-in
- **WHEN** a signed-in operator signs out and then opens a console route
- **THEN** they are redirected to the sign-in flow

#### Scenario: A replayed session id is dead
- **WHEN** a session-id cookie value captured before sign-out is replayed on a console route
- **THEN** the request is treated as unauthenticated and redirected to sign-in

### Requirement: Application roles are AAS tenant roles

The two application roles `dispatch-viewer` and `dispatch-dispatcher` SHALL be declared
as AAS tenant roles of the `development` tenant in
`compose/iams/init/project-aas/roles.yaml`, and SHALL be surfaced to services in the
JWT's `active_tenant.roles` claim. They SHALL NOT be Keycloak realm roles.

#### Scenario: A seeded operator carries the dispatcher role
- **WHEN** a token is obtained for the seeded dispatcher user via the password grant against the bundled `web` client on the `aoh` realm, and decoded
- **THEN** `active_tenant.roles` contains `dispatch-dispatcher`
- **AND** `active_tenant.tenant_id` is the `development` tenant's id

#### Scenario: A seeded viewer carries only the viewer role
- **WHEN** a token is obtained the same way for the seeded viewer user and decoded
- **THEN** `active_tenant.roles` contains `dispatch-viewer` and does not contain `dispatch-dispatcher`

#### Scenario: The role bootstrap is idempotent
- **WHEN** `project-aas-init` runs twice against the same stack
- **THEN** the role set after the second run is identical to the set after the first

#### Scenario: Roles are absent from the realm import
- **WHEN** `compose/iams/keycloak/realm-import.json` is searched for `dispatch-viewer` or `dispatch-dispatcher`
- **THEN** neither name appears

### Requirement: Permissions are projected from roles inside each process

The console and `dispatch-svc` SHALL each derive their permissions from the role names in
`active_tenant.roles` using a static projection that mirrors `roles.yaml`. Neither SHALL
read a resolved permission list from the token, because AAS does not put one there.

#### Scenario: The projection is what gates the action
- **WHEN** the authorization path in either app is reviewed
- **THEN** it reads `active_tenant.roles` and maps role names to permissions in-process
- **AND** it references no `active_tenant.permissions` claim

#### Scenario: A token with an unknown role is not granted write access
- **WHEN** a caller presents a valid token whose `active_tenant.roles` contains neither application role
- **THEN** a write is refused with 403

### Requirement: The field-unit API requires a bearer token

Every `/v1/units` endpoint SHALL require a valid bearer token issued by the `aoh` realm,
validated offline against the realm's JWKS. `/livez` and `/readyz` SHALL remain
unauthenticated.

#### Scenario: A request with no token is rejected
- **WHEN** a client issues `GET /v1/units` with no `Authorization` header
- **THEN** the response status is 401
- **AND** no unit data is returned

#### Scenario: A request with an invalid or expired token is rejected
- **WHEN** a client issues `GET /v1/units` with a malformed, wrongly signed, or expired token
- **THEN** the response status is 401

#### Scenario: A valid token is accepted
- **WHEN** a client issues `GET /v1/units` with a token obtained via the password grant against the bundled `web` client
- **THEN** the response status is 200

### Requirement: Health probes stay outside the authenticated surface

Both applications' `GET /livez` and `GET /readyz` SHALL remain reachable with no session and
no token, mounted at the root. Restoring authentication SHALL NOT put them behind the
sign-in redirect or the bearer middleware — Kubernetes probes them without credentials.

#### Scenario: The service's probes stay open
- **WHEN** `GET /livez` and `GET /readyz` are issued to `dispatch-svc` with no `Authorization` header
- **THEN** both answer as they did before this change, with no 401

#### Scenario: The console's probes stay open
- **WHEN** `GET /livez` and `GET /readyz` are issued to `dispatch-web` with no session cookie
- **THEN** both answer 200 and are not redirected to the sign-in flow

### Requirement: Writing a unit requires the dispatcher role

Creating, replacing and deleting a field unit SHALL require `dispatch-dispatcher` in the
caller's `active_tenant.roles`. Reading SHALL require only a valid token with either
application role. The console SHALL NOT render a write control an operator's roles do not
permit, and the service SHALL enforce the same rule independently.

#### Scenario: A viewer cannot write
- **WHEN** a caller holding only `dispatch-viewer` issues `POST /v1/units`, `PUT /v1/units/{unit_code}` or `DELETE /v1/units/{unit_code}`
- **THEN** the response status is 403
- **AND** the stored unit is unchanged

#### Scenario: A dispatcher can write
- **WHEN** a caller holding `dispatch-dispatcher` issues the same requests with a valid body
- **THEN** the response succeeds as specified by `dispatch-units-api`

#### Scenario: A viewer sees no write controls
- **WHEN** an operator holding only `dispatch-viewer` opens the console
- **THEN** the add, edit and delete controls are absent
- **AND** the roster and detail pane render normally

#### Scenario: The service enforces the rule even when the console does not
- **WHEN** a viewer's browser submits a write directly to the console's form action
- **THEN** the write is refused and the console shows a permission-denied message rather than a generic failure
