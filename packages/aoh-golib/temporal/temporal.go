package temporal

import (
	"context"

	log "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/temporal/propagator"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

type Config struct {
	HostPort  string
	Namespace string
}

type Client = client.Client

// NewClient creates client connection to temporal server
func NewClient(c Config) (Client, error) {
	tempLogger := log.Get()
	tempLogger.WithOptions(zap.AddCallerSkip(1))

	cli, err := client.Dial(client.Options{
		HostPort:           c.HostPort,
		Namespace:          c.Namespace,
		Logger:             logur.LoggerToKV(zapadapter.New(tempLogger)),
		ContextPropagators: []workflow.ContextPropagator{propagator.NewContextPropagator()},
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// ContextWithValue sets temporal context value
func ContextWithValue(ctx context.Context, value map[string]interface{}) context.Context {
	return context.WithValue(ctx, propagator.PropagateKey, value)
}

// ContextValue returns temporal context value set by ContextWithValue
func ContextValue(ctx context.Context) map[string]interface{} {
	if val := ctx.Value(propagator.PropagateKey); val != nil {
		vals := val.(map[string]interface{})
		return vals
	}
	return nil
}
