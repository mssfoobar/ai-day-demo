## ADDED Requirements

### Requirement: Operators sign in through IAMS

The console SHALL authenticate operators through `iams-keycloak` using OpenID Connect
Authorization Code flow with PKCE against the bundled `aoh` realm. It SHALL reuse the
realm's existing public `web` client. This change SHALL NOT create a realm, SHALL NOT
register an OIDC client of any kind, SHALL NOT add a realm role, and SHALL NOT add a claim
mapper. Its only addition to `realm-import.json` SHALL be one seed user (see below).

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
- **THEN** they end at the sign-in flow, not at an unauthenticated console page and not at a 404

#### Scenario: The scaffold's auth routes are restored as a set
- **WHEN** the app's `(public)/aoh/api/auth/` directory is listed
- **THEN** it contains all six route handlers the scaffold ships — `login`, `callback`, `refresh`, `logout`, `context` and `context/[value]`
- **AND** requesting the tenant-switching routes as a signed-in operator returns without a server error, even though this change adds no second tenant to switch to

#### Scenario: The change adds no client, role or mapper
- **WHEN** `compose/iams/keycloak/realm-import.json` is compared against the platform-shipped file
- **THEN** the only difference is one added entry in the `users` array
- **AND** the `clients`, realm-roles and claim-mapper sections are byte-identical

### Requirement: The realm seeds one viewer account alongside the existing operator

This change SHALL add one seed user to `realm-import.json`'s `users` array to serve as the
viewer account, and SHALL assign the two application roles through `roles.yaml`. The shipped
`aoh` realm contains exactly one interactive user, so the viewer/dispatcher split has no
second account to assign otherwise. The role bootstrap SHALL NOT be expected to create
the account: `project-aas-init` requires a user that already exists in Keycloak and exits
with an error otherwise.

#### Scenario: Both accounts exist after a fresh import
- **WHEN** the stack is brought up from empty volumes
- **THEN** both the dispatcher account and the viewer account can complete the sign-in flow
- **AND** neither required a manual step in the Keycloak admin UI

#### Scenario: The role bootstrap finds both users
- **WHEN** `project-aas-init` runs
- **THEN** it exits 0
- **AND** it does not report a user missing from Keycloak

### Requirement: The projection carries the operator's bearer

The outbox worker SHALL authenticate to `gis-service` with the access token of the operator
whose write produced the outbox row, captured by value before the post-commit work detaches
from the request context. It SHALL NOT use a client-credentials (service-account) token,
because such a token carries no `active_tenant` claim and `gis-service` resolves the tenant
from that claim. No access token, refresh token, or client secret SHALL be persisted in the
outbox or anywhere else in the database.

#### Scenario: The projection is made as the operator
- **WHEN** the worker delivers a projection for a unit a dispatcher just wrote
- **THEN** the request to `gis-service` carries that dispatcher's access token
- **AND** `gis-service` accepts it

#### Scenario: The outbox holds no credentials
- **WHEN** an outbox row is inspected
- **THEN** it carries the target unit, the intent and the payload
- **AND** it carries no access token, refresh token, or client secret

#### Scenario: A delivery that outlives its token is left pending, not reassigned
- **WHEN** a projection cannot be delivered before the writing operator's token expires
- **THEN** the outbox row remains pending and is not marked delivered
- **AND** no later request by a different operator is used to deliver it

#### Scenario: A reader is never made to perform a write
- **WHEN** an operator holding only `dispatch-viewer` reads the roster while a projection is pending
- **THEN** no write to `gis-service` is made with that operator's token

#### Scenario: Tokens are requested with the openid scope
- **WHEN** any component of this change obtains a token for calling an AOH service
- **THEN** the request includes `scope=openid`
- **AND** the resulting token is accepted rather than rejected as an invalid JWT

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

### Requirement: A session outlives the access token without the operator noticing

Access tokens are short-lived. The session SHALL survive their expiry: the console SHALL
renew through its refresh endpoint, backed by the refresh token held in SDS, without
returning the operator to the sign-in screen and without the browser ever holding either
token.

#### Scenario: Working across an access-token expiry
- **WHEN** an operator leaves the console idle for longer than the access token's lifetime and then performs a read or a write
- **THEN** the action succeeds
- **AND** they are not redirected to the sign-in flow

#### Scenario: Renewal happens server-side
- **WHEN** a renewal occurs
- **THEN** the browser's cookies still carry only the session id
- **AND** no access or refresh token appears in any response to the browser

#### Scenario: An ended session cannot be renewed
- **WHEN** renewal is attempted for a session that has been signed out
- **THEN** it fails and the operator is sent to the sign-in flow

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
validated by `aoh-golib`'s shipped `aohhttp.BearerAuth` middleware rather than a hand-rolled
equivalent. This includes the not-yet-implemented workshop stub routes. `/livez` and
`/readyz` SHALL remain unauthenticated.

#### Scenario: A request with no token is rejected
- **WHEN** a client issues `GET /v1/units` with no `Authorization` header
- **THEN** the response status is 401
- **AND** no unit data is returned

#### Scenario: A request with an invalid or expired token is rejected
- **WHEN** a client issues `GET /v1/units` with a malformed token, or with a token that has passed its expiry
- **THEN** the response status is 401

#### Scenario: A valid token is accepted
- **WHEN** a client issues `GET /v1/units` with a token obtained via the password grant against the bundled `web` client, requested with `scope=openid`
- **THEN** the response status is 200

#### Scenario: The unimplemented stub routes are also protected
- **WHEN** a client issues any of `POST /v1/units/{unit_code}/assignment`, `DELETE /v1/units/{unit_code}/assignment`, `GET /v1/units/{unit_code}/events` or `PUT /v1/units/{unit_code}/crew` with no `Authorization` header
- **THEN** the response status is 401
- **AND** the 501 body those routes return to an authorised caller is not disclosed

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
