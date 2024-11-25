package quic

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"zombiezen.com/go/capnproto2/rpc"

	"github.com/google/uuid"

	"github.com/khulnasoft/tunnellink/tunnelrpc"
	"github.com/khulnasoft/tunnellink/tunnelrpc/metrics"
	"github.com/khulnasoft/tunnellink/tunnelrpc/pogs"
)

// TunnellinkClient calls capnp rpc methods of SessionManager and ConfigurationManager.
type TunnellinkClient struct {
	client         pogs.TunnellinkServer_PogsClient
	transport      rpc.Transport
	requestTimeout time.Duration
}

func NewTunnellinkClient(ctx context.Context, stream io.ReadWriteCloser, requestTimeout time.Duration) (*TunnellinkClient, error) {
	n, err := stream.Write(rpcStreamProtocolSignature[:])
	if err != nil {
		return nil, err
	}
	if n != len(rpcStreamProtocolSignature) {
		return nil, fmt.Errorf("expect to write %d bytes for RPC stream protocol signature, wrote %d", len(rpcStreamProtocolSignature), n)
	}
	transport := tunnelrpc.SafeTransport(stream)
	conn := tunnelrpc.NewClientConn(transport)
	client := pogs.NewTunnellinkServer_PogsClient(conn.Bootstrap(ctx), conn)
	return &TunnellinkClient{
		client:         client,
		transport:      transport,
		requestTimeout: requestTimeout,
	}, nil
}

func (c *TunnellinkClient) RegisterUdpSession(ctx context.Context, sessionID uuid.UUID, dstIP net.IP, dstPort uint16, closeIdleAfterHint time.Duration, traceContext string) (*pogs.RegisterUdpSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	defer metrics.CapnpMetrics.ClientOperations.WithLabelValues(metrics.Tunnellink, metrics.OperationRegisterUdpSession).Inc()
	timer := metrics.NewClientOperationLatencyObserver(metrics.Tunnellink, metrics.OperationRegisterUdpSession)
	defer timer.ObserveDuration()

	resp, err := c.client.RegisterUdpSession(ctx, sessionID, dstIP, dstPort, closeIdleAfterHint, traceContext)
	if err != nil {
		metrics.CapnpMetrics.ClientFailures.WithLabelValues(metrics.Tunnellink, metrics.OperationRegisterUdpSession).Inc()
	}
	return resp, err
}

func (c *TunnellinkClient) UnregisterUdpSession(ctx context.Context, sessionID uuid.UUID, message string) error {
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	defer metrics.CapnpMetrics.ClientOperations.WithLabelValues(metrics.Tunnellink, metrics.OperationUnregisterUdpSession).Inc()
	timer := metrics.NewClientOperationLatencyObserver(metrics.Tunnellink, metrics.OperationUnregisterUdpSession)
	defer timer.ObserveDuration()

	err := c.client.UnregisterUdpSession(ctx, sessionID, message)
	if err != nil {
		metrics.CapnpMetrics.ClientFailures.WithLabelValues(metrics.Tunnellink, metrics.OperationUnregisterUdpSession).Inc()
	}
	return err
}

func (c *TunnellinkClient) UpdateConfiguration(ctx context.Context, version int32, config []byte) (*pogs.UpdateConfigurationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	defer metrics.CapnpMetrics.ClientOperations.WithLabelValues(metrics.Tunnellink, metrics.OperationUpdateConfiguration).Inc()
	timer := metrics.NewClientOperationLatencyObserver(metrics.Tunnellink, metrics.OperationUpdateConfiguration)
	defer timer.ObserveDuration()

	resp, err := c.client.UpdateConfiguration(ctx, version, config)
	if err != nil {
		metrics.CapnpMetrics.ClientFailures.WithLabelValues(metrics.Tunnellink, metrics.OperationUpdateConfiguration).Inc()
	}
	return resp, err
}

func (c *TunnellinkClient) Close() {
	_ = c.client.Close()
	_ = c.transport.Close()
}
