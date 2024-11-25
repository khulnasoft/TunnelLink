package pogs

import (
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	"github.com/khulnasoft/tunnellink/tunnelrpc/proto"
)

type TunnellinkServer interface {
	SessionManager
	ConfigurationManager
}

type TunnellinkServer_PogsImpl struct {
	SessionManager_PogsImpl
	ConfigurationManager_PogsImpl
}

func TunnellinkServer_ServerToClient(s SessionManager, c ConfigurationManager) proto.TunnellinkServer {
	return proto.TunnellinkServer_ServerToClient(TunnellinkServer_PogsImpl{
		SessionManager_PogsImpl:       SessionManager_PogsImpl{s},
		ConfigurationManager_PogsImpl: ConfigurationManager_PogsImpl{c},
	})
}

type TunnellinkServer_PogsClient struct {
	SessionManager_PogsClient
	ConfigurationManager_PogsClient
	Client capnp.Client
	Conn   *rpc.Conn
}

func NewTunnellinkServer_PogsClient(client capnp.Client, conn *rpc.Conn) TunnellinkServer_PogsClient {
	sessionManagerClient := SessionManager_PogsClient{
		Client: client,
		Conn:   conn,
	}
	configManagerClient := ConfigurationManager_PogsClient{
		Client: client,
		Conn:   conn,
	}
	return TunnellinkServer_PogsClient{
		SessionManager_PogsClient:       sessionManagerClient,
		ConfigurationManager_PogsClient: configManagerClient,
		Client:                          client,
		Conn:                            conn,
	}
}

func (c TunnellinkServer_PogsClient) Close() error {
	c.Client.Close()
	return c.Conn.Close()
}
