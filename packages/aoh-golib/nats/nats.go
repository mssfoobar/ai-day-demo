package aohnats

import (
	"context"
	"time"

	"go.uber.org/zap"

	log "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
)

const (
	retryCount int           = 3
	retryDelay time.Duration = time.Second
)

// NewNatsConn will attempt to connect to nats server with given url and opts.
//
// Following nats options are already preconfigured for common use cases
//   - nats.ReconnectWait
//   - nats.MaxReconnects
//   - nats.DisconnectErrHandler
func NewNatsConn(url string, opts []nats.Option) (*nats.Conn, error) {
	opts = setupConnOptions(opts)

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		if !nc.IsClosed() {
			log.Warn("Disconnected", zap.String("reason", err.Error()),
				zap.Duration("wait", time.Duration(totalWait.Minutes())))
		}
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Info("Reconnected", zap.String("url", nc.ConnectedUrl()))
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		if !nc.IsClosed() {
			log.Error("Exiting: no servers available")
		} else {
			log.Error("Exiting")
		}
	}))
	return opts
}

// RegisterService for service discovery with 'name' and 'version' of the service
func RegisterService(
	nc *nats.Conn,
	name string,
	version string,
) (*micro.Service, error) {
	handler := func(req micro.Request) {
		err := req.Respond(req.Data())
		if err != nil {
			log.Error(err.Error())
		}
	}

	config := micro.Config{
		Name:    name,
		Version: version,
		Endpoint: &micro.EndpointConfig{
			Subject: "service.discovery",
			Handler: micro.HandlerFunc(handler),
		},
	}
	srv, err := micro.AddService(nc, config)
	if err != nil {
		return nil, err
	}

	return &srv, nil
}

// CreateConsumer create durable nats consumer with config - DeliverAllPolicy and AckExplicitPolicy.
func CreateConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	name string,
	subject string,
	stream string,
) (jetstream.Consumer, error) {
	config := jetstream.ConsumerConfig{
		Durable:       name,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: subject + ".*",
	}

	var cons jetstream.Consumer
	var err error
	for retry := 0; retry < retryCount; retry++ {
		cons, err = js.CreateConsumer(ctx, stream, config)
		if err == nil {
			break
		}
		time.Sleep(retryDelay)
	}
	if err != nil {
		return nil, err
	}

	return cons, nil
}
