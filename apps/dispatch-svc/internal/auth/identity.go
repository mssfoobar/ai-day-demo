// Package auth carries the caller's identity from the bearer token onto the request
// context, so the layers below never parse a JWT themselves.
//
// Validation is not this package's job: `aohhttp.BearerAuth` has already checked the
// token against Keycloak's userinfo endpoint by the time Identify runs. What is left is
// reading the claims the service needs — who the caller is, which tenant they are acting
// in, which application roles they hold — and keeping the raw token so an outbound call
// can be made as them (design.md D2a).
package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
)

// Identity is the caller, as the token describes them.
//
// BearerToken is held by value, not read back off the request, because the projection
// worker outlives the request: a post-commit goroutine that reached for the token
// through the request context would find it cancelled (design.md D2a). It is never
// persisted — not in the outbox, not anywhere.
type Identity struct {
	Subject     string
	Name        string
	TenantID    string
	Roles       []string
	BearerToken string
	// ExpiresAt is the token's `exp`. It bounds how long a projection may be retried:
	// past it the token is dead and the row stays pending rather than being delivered
	// on someone else's credential.
	ExpiresAt time.Time
}

type contextKey struct{}

// WithIdentity returns a context carrying the caller.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the caller on the context, if Identify ran.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(contextKey{}).(Identity)
	return id, ok
}

// CodeIdentityMissing — the request reached a handler with no identity on its context.
// Only reachable if the middleware chain is mis-wired, so it is a system fault, not a
// client one.
var CodeIdentityMissing = aoherr.MustCode("DISPATCH_IDENTITY_MISSING")

// MustFromContext returns the caller, or a classified system error naming the wiring bug.
func MustFromContext(ctx context.Context) (Identity, error) {
	id, ok := FromContext(ctx)
	if !ok {
		return Identity{}, aoherr.New(aoherr.ClassSystem, CodeIdentityMissing,
			"the request carries no caller identity")
	}
	return id, nil
}

// CodeTenantMissing — the token is valid but carries no active tenant, so there is no
// tenant to scope the request to. A client-credentials token looks exactly like this,
// which is why this change never uses one.
var CodeTenantMissing = aoherr.MustCode("DISPATCH_TENANT_MISSING")

// Identify reads the validated bearer token's claims onto the request context.
//
// Mount it *after* aohhttp.BearerAuth: it re-parses the same token without verifying it,
// which is only safe because BearerAuth has already rejected anything invalid.
func Identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claim, err := aohhttp.GetJWTClaim(r)
		if err != nil {
			// GetJWTClaim's only failure past BearerAuth is an absent active_tenant —
			// a well-formed token that cannot be acted on, which is a 401 about the
			// credential rather than a 403 about the caller's roles.
			aoherr.Render(w, r, aoherr.Wrap(aoherr.ClassAuthentication, CodeTenantMissing,
				"the bearer token carries no active tenant", err))
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		id := Identity{
			Subject:     claim.Id,
			Name:        claim.Name,
			TenantID:    claim.ActiveTenant.Id,
			Roles:       claim.ActiveTenant.Roles,
			BearerToken: token,
			ExpiresAt:   expiryOf(token),
		}
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), id)))
	})
}

// expiryOf reads the token's `exp` without verifying it — BearerAuth already did that.
// A token whose expiry cannot be read yields the zero time, which the projection worker
// treats as "no budget" rather than "unlimited".
func expiryOf(token string) time.Time {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return time.Time{}
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return time.Time{}
	}
	return exp.Time
}
