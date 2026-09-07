package aohgraphql

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	log "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"go.uber.org/zap"

	"github.com/Nerzal/gocloak/v13"
)

const (
	retryCount int           = 3
	retryDelay time.Duration = time.Second
	jwtExpired string        = "JWTExpired"
)

var token *gocloak.JWT

// Client is the wrapper of genqlient/graphql.
// Unbuffered refreshToken channel is used to auto refresh JWT access token.
type Client struct {
	graphql.Client
	refreshToken chan struct{}
}

type Config struct {
	Address         string
	IamUrl          string
	IamClientId     string
	IamClientSecret string
	IamRealm        string
}

func getToken(c Config) (*gocloak.JWT, error) {
	client := gocloak.NewClient(c.IamUrl)
	ctx := context.Background()

	grant := "client_credentials"
	token, err := client.GetToken(ctx, c.IamRealm, gocloak.TokenOptions{
		ClientID:     &c.IamClientId,
		ClientSecret: &c.IamClientSecret,
		GrantType:    &grant,
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

type authedTransport struct {
	wrapped http.RoundTripper
	conf    Config
}

type jwtExpire struct {
	Errors []struct {
		Message string
	}
}

// isJwtExpireError checks http response body message and return true if the message contains jwtExpired.
func isJwtExpireError(res *http.Response) bool {
	b, err := io.ReadAll(res.Body)
	defer func() {
		res.Body = io.NopCloser(bytes.NewBuffer(b))
	}()
	if err != nil {
		return false
	}
	var jsonBody jwtExpire
	if err := json.Unmarshal(b, &jsonBody); err != nil {
		return false
	}
	if len(jsonBody.Errors) == 0 {
		return false
	}
	if strings.Contains(jsonBody.Errors[0].Message, jwtExpired) {
		return true
	}
	return false
}

// RoundTrip implements http/client RoundTripper.
// If JWT token is expired, it will try to refresh the token before trying again.
func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var res *http.Response
	var err error
	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	res, err = t.wrapped.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if isJwtExpireError(res) {
		for i := 0; i < retryCount; i++ {
			token, err = getToken(t.conf)
			if err != nil {
				continue
			}
			req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
			res, err = t.wrapped.RoundTrip(req)
			if err == nil {
				break
			}
			time.Sleep(retryDelay)
		}
	}
	return res, nil
}

// NewGraphqlClient returns graphql client connection with built-in access token refresh.
func NewGraphqlClient(c Config) *Client {
	var err error
	token, err = getToken(c)
	if err != nil {
		log.Fatal(err.Error())
	}

	ch := make(chan struct{})
	client := Client{
		refreshToken: ch,
	}

	client.Client = graphql.NewClient(
		c.Address,
		&http.Client{
			Transport: &authedTransport{
				wrapped: http.DefaultTransport,
				conf:    c,
			},
		})

	// Start a goroutine to periodically refresh the token
	go func() {
		for {
			minimumInterval := 10.0
			leewayInSeconds := 30.0
			timeBeforeRefresh := time.Duration(
				math.Max(float64(token.ExpiresIn)-leewayInSeconds, minimumInterval),
			) * time.Second

			// Wait before refreshing the token again
			log.Info(
				"Sleep before token refresh",
				zap.Float64("seconds", timeBeforeRefresh.Seconds()),
			)
			time.Sleep(timeBeforeRefresh)
			token, err = getToken(c)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
	}()
	return &client
}
