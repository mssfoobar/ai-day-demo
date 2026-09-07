package aohhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrEmptyTenant = errors.New("empty active tenant")
var ErrInvalidJWT = errors.New("invalid jwt")
var ErrUnexpectedStatusCode = errors.New("unexpected status code")

type JwtClaim struct {
	Id           string `json:"sub"`
	Name         string `json:"name"`
	ActiveTenant struct {
		Id    string   `json:"tenant_id"`
		Name  string   `json:"tenant_name"`
		Roles []string `json:"roles"`
	} `json:"active_tenant"`
	ResourceAccess map[string]any `json:"resource_access"`
}

// GetJWTClaim extracts the JWT from the request.
// This parses the jwt into a JwtClaim struct and checks if it has active tenant id.
// Return error on invalid JwtClaim format or empty active tenant id.
//
// If the jwt is not in JwtClaim format, use ParseJWT instead.
func GetJWTClaim(r *http.Request) (*JwtClaim, error) {
	token, err := ParseJWT(r)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}

	var claim JwtClaim
	err = json.Unmarshal(raw, &claim)
	if err != nil {
		return nil, err
	}

	if claim.ActiveTenant.Id == "" {
		return nil, ErrEmptyTenant
	}

	return &claim, nil
}

// ParseJWT extracts the JWT from the request and parses it into a map[string]any.
func ParseJWT(r *http.Request) (map[string]any, error) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return parseJWT(token)
}

// ValidateJWTWithRetry checks if the jwt is valid by using the discovered userinfo endpoint via well-known config.
// If the well-known config is changed, it retries with refreshed well-known config.
//
// issuerUrl is the url of the issuer, e.g. http://iams-keycloak:8080/realms/AOH
func ValidateJWTWithRetry(issuerUrl string, token string) error {
	for attempt := 0; attempt < 2; attempt++ {
		config, err := GetOpenIDCache().GetWellKnownConfig(issuerUrl)
		if err != nil {
			return err
		}

		valid, err := isValidJWT(config.UserinfoEndpoint, token)
		if err != nil {
			// unexpected status code is retried with refreshed well-known config
			// this can happen if the well-known config is changed
			if errors.Is(err, ErrUnexpectedStatusCode) {
				if err := GetOpenIDCache().DiscoverWellKnownConfig(issuerUrl); err != nil {
					return err
				}
				continue
			}
			return err
		}

		if valid {
			return nil
		} else {
			return ErrInvalidJWT
		}
	}

	return ErrInvalidJWT
}

func parseJWT(token string) (map[string]any, error) {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(token, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwt: %w", err)
	}
	return claims, nil
}

func isValidJWT(userinfoEndpoint string, token string) (bool, error) {
	parsed, err := parseJWT(token)
	if err != nil {
		return false, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	r, err := http.NewRequest("GET", userinfoEndpoint, nil)
	if err != nil {
		return false, err
	}
	r.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(r)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() //nolint:errcheck

	// validation endpoint returns only 200 or 401
	// others status codes are unexpected
	if resp.StatusCode == 401 {
		return false, nil
	} else if resp.StatusCode != 200 {
		return false, ErrUnexpectedStatusCode
	}

	// check if the sub claim matches
	var claim JwtClaim
	if err = json.NewDecoder(resp.Body).Decode(&claim); err != nil {
		return false, err
	}
	// Checked, not asserted. `parsed["sub"].(string)` panicked on any token
	// without a sub claim — attacker-controlled input on the authentication path
	// of every service. aohhttp.Recoverer turned that into a 500, so a malformed
	// token was reported as a server fault rather than as a rejected credential,
	// and each one cost a panic/recover.
	sub, ok := parsed["sub"].(string)
	if !ok || claim.Id != sub {
		return false, nil
	}

	return true, nil
}
