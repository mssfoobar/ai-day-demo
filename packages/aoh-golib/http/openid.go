package aohhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const openIDWellKnownConfigPath string = "/.well-known/openid-configuration"

var (
	openIDCache     *OpenIDCache
	openIDCacheOnce sync.Once
)

// OpenIDWellKnownConfig represents the OpenID well-known configuration.
// Only UserinfoEndpoint is used to validate tokens for now.
type OpenIDWellKnownConfig struct {
	// endpoints
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	IntrospectionEndpoint string `json:"introspection_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`

	// Supported features
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
}

type openIDEntry struct {
	config        OpenIDWellKnownConfig
	lastRefreshed time.Time
}

// OpenIDCache holds OpenID configurations and manages their refresh lifecycle.
//
// Entries are keyed BY ISSUER URL. They were not always: this cache used to hold
// a single config for the whole process regardless of the issuerUrl argument, so
// the first discovery won and a second issuer in the same process silently got
// the first one's endpoints. Nothing in AOH runs two issuers today, but the
// package's own tests had to document the flaw and work around it ("one issuer,
// one discovery"), which is a fair sign it would eventually bite something real.
type OpenIDCache struct {
	entries  map[string]*openIDEntry
	duration time.Duration
	mu       sync.RWMutex
}

// GetOpenIDCache returns the singleton instance of OpenIDCache.
func GetOpenIDCache() *OpenIDCache {
	openIDCacheOnce.Do(func() {
		openIDCache = &OpenIDCache{
			entries:  map[string]*openIDEntry{},
			duration: 24 * time.Hour,
		}
	})

	return openIDCache
}

// GetWellKnownConfig returns the OpenID well-known configuration for the given issuer URL.
func (c *OpenIDCache) GetWellKnownConfig(issuerUrl string) (*OpenIDWellKnownConfig, error) {
	c.mu.RLock()
	entry, ok := c.entries[issuerUrl]
	var config OpenIDWellKnownConfig
	expired := true
	if ok {
		config = entry.config
		expired = time.Since(entry.lastRefreshed) > c.duration
	}
	c.mu.RUnlock()

	if ok && config.UserinfoEndpoint != "" && !expired {
		return &config, nil
	}

	if err := c.DiscoverWellKnownConfig(issuerUrl); err != nil {
		return nil, fmt.Errorf("failed to discover openid well-known configuration: %w", err)
	}

	c.mu.RLock()
	config = c.entries[issuerUrl].config
	c.mu.RUnlock()

	return &config, nil
}

// DiscoverWellKnownConfig refreshes the OpenID well-known configuration for the given issuer URL.
func (c *OpenIDCache) DiscoverWellKnownConfig(issuerUrl string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(issuerUrl + openIDWellKnownConfigPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code: %d; response body: %s", resp.StatusCode, string(body))
	}

	var config OpenIDWellKnownConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	if c.entries == nil {
		c.entries = map[string]*openIDEntry{}
	}
	c.entries[issuerUrl] = &openIDEntry{config: config, lastRefreshed: time.Now()}

	return nil
}
