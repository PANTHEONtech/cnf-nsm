/*
 * Copyright (c) 2020 PANTHEON.tech s.r.o. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package descriptor

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"go.ligato.io/cn-infra/v2/datasync"
	"go.ligato.io/cn-infra/v2/logging"
	"go.ligato.io/vpp-agent/v3/client"
	kvs "go.ligato.io/vpp-agent/v3/plugins/kvscheduler/api"
	linux_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/linux/interfaces"
	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	"go.cdnf.io/cnf-nsm/plugins/nsmplugin/descriptor/adapter"
	"go.cdnf.io/cnf-nsm/proto/nsm"

	nsm_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	nsm_common "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	nsm_memif "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	nsm_registry "github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	nsm_sdk_tools "github.com/networkservicemesh/networkservicemesh/pkg/tools"
	nsm_span "github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	nsm_sdk_common "github.com/networkservicemesh/networkservicemesh/sdk/common"
	nsm_sdk_endpoint "github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

const (
	nsmEndopintDescriptorName = "nsm-endpoint-descriptor"
)

// validation errors
var (
	// ErrEndpointWithoutName is returned when NSM endpoint is configured with undefined name.
	ErrEndpointWithoutName = errors.New("NSM endpoint defined without logical name")

	// ErrEndpointWithoutNetworkService is returned when NSM endpoint is configured with undefined network service name.
	ErrEndpointWithoutNetworkService = errors.New("NSM endpoint defined without network service name")

	// ErrEndpointWithoutInterface is returned when NSM endpoint is configured with undefined interface name prefix.
	ErrEndpointWithoutInterface = errors.New("NSM endpoint defined without interface name prefix")
)

// NSMEndpointDescriptor creates NSM endpoints.
type NSMEndpointDescriptor struct {
	// dependencies
	log            logging.Logger
	localclient    client.ConfigClient
	notifPublisher NotificationPublisher
	epDemux        *NSMEndpointDemux
}

// NotificationPublisher is an optional dependency of the NSM Endpoint descriptor which, if injected, is used
// to publish notifications about registered endpoints.
type NotificationPublisher interface {
	Put(key string, data proto.Message, opts ...datasync.PutOption) error
	Delete(key string, opts ...datasync.DelOption) (existed bool, err error)
}

// NSMEndpoint implements the Network service endpoint functionality needed for NSM plugin.
// Most importantly it creates the connection interface with the requested configuration.
type NSMEndpoint struct {
	log         logging.Logger
	localclient client.ConfigClient
	cfg         *nsm.NetworkServiceEndpoint
	connections map[string]nsmConnection
}

// NSMEndpointDemux routes connection request to the corresponding network service.
type NSMEndpointDemux struct {
	sync.Mutex
	log         logging.Logger
	connected   bool
	grpcServer  *grpc.Server
	nsmConn     *nsm_sdk_common.NsmConnection
	nsmRegistry nsm_registry.NetworkServiceRegistryClient
	endpoints   map[string]*endpointReg // key = network service name
}

// Endpoint registration.
type endpointReg struct {
	EpName  string
	Service networkservice.NetworkServiceServer
	Close   func() error
}

// Connection between NSM client and NSM endpoint.
type nsmConnection struct {
	index int
	conn  *nsm_connection.Connection
}

// NewNSMEndpointDescriptor creates new instance of the descriptor.
func NewNSMEndpointDescriptor(log logging.PluginLogger, localclient client.ConfigClient, notifPublisher NotificationPublisher) (
	descrCtx *NSMEndpointDescriptor, descr *kvs.KVDescriptor) {

	descrCtx = &NSMEndpointDescriptor{
		log:            log.NewLogger(nsmEndopintDescriptorName),
		localclient:    localclient,
		notifPublisher: notifPublisher,
		epDemux:        NewNSMEndpointDemux(log.NewLogger("nsm-endpoint-demux")),
	}
	typedDescr := &adapter.NSMEndpointDescriptor{
		Name:          nsmEndopintDescriptorName,
		KeySelector:   nsm.ModelNSMEndpoint.IsKeyValid,
		ValueTypeName: nsm.ModelNSMEndpoint.ProtoName(),
		KeyLabel:      nsm.ModelNSMEndpoint.StripKeyPrefix,
		NBKeyPrefix:   nsm.ModelNSMEndpoint.KeyPrefix(),
		WithMetadata:  true,
		Validate:      descrCtx.Validate,
		Create:        descrCtx.Create,
		Delete:        descrCtx.Delete,
	}
	return descrCtx, adapter.NewNSMEndpointDescriptor(typedDescr)
}

// NewNSMEndpoint creates an instance of NSMEndpoint.
func NewNSMEndpoint(cfg *nsm.NetworkServiceEndpoint, log logging.Logger,
	localclient client.ConfigClient) *NSMEndpoint {

	return &NSMEndpoint{
		cfg:         cfg,
		log:         log,
		localclient: localclient,
		connections: make(map[string]nsmConnection),
	}
}

// NewNSMEndpoint creates an instance of NSMEndpointDemux.
func NewNSMEndpointDemux(log logging.Logger) *NSMEndpointDemux {
	return &NSMEndpointDemux{
		log:       log,
		endpoints: make(map[string]*endpointReg),
	}
}

// Validate validates NSM-Endpoint configuration.
func (d *NSMEndpointDescriptor) Validate(key string, cfg *nsm.NetworkServiceEndpoint) error {
	if cfg.GetNetworkService() == "" {
		return kvs.NewInvalidValueError(ErrEndpointWithoutNetworkService, "network_service")
	}
	if cfg.GetInterfaceNamePrefix() == "" {
		return kvs.NewInvalidValueError(ErrEndpointWithoutInterface, "interface_name_prefix")
	}
	return nil
}

// Creates announces new endpoint within a given network service.
func (d *NSMEndpointDescriptor) Create(key string, cfg *nsm.NetworkServiceEndpoint) (metadata interface{}, err error) {
	// prepare NSM endpoint configuration
	configuration := nsm_sdk_common.FromEnv()
	configuration.MechanismType = nsmMechanismFrominterfaceType(cfg.GetInterfaceType())
	configuration.AdvertiseNseName = cfg.GetNetworkService()
	configuration.AdvertiseNseLabels = nsmlabelsFromProto(cfg.GetAdvertisedLabels())

	// Currently NSM is unable/refuses to create IP-less connections (SrcIpRequired & DstIpRequired are always enabled
	// by NsmClient.Connect). Therefore we generate a dummy subnet from within the reserved IP address block 0.0.0.0/8
	// that is unlikely to collide with subnets generated for other clients/endpoints of this agent instance.
	configuration.IPAddress = generateDummySubnet("endpoint-" + cfg.GetNetworkService())

	// build NSM endpoint using composite
	ep := NewNSMEndpoint(cfg, d.log, d.localclient)
	closer := func() error {
		return ep.CloseConnections()
	}
	compositeEndpoints := []networkservice.NetworkServiceServer{
		nsm_sdk_endpoint.NewMonitorEndpoint(configuration),
		nsm_sdk_endpoint.NewConnectionEndpoint(configuration),
		nsm_sdk_endpoint.NewIpamEndpoint(configuration), // not actually used for now
		ep,
	}
	composite := nsm_sdk_endpoint.NewCompositeEndpoint(compositeEndpoints...)

	// Register endpoint with the demultiplexer.
	err = d.epDemux.AddEndpoint(context.Background(), configuration, composite, closer)
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	// publish notification about the newly registered endpoint
	if d.notifPublisher != nil {
		for _, label := range cfg.AdvertisedLabels {
			key := nsm.EndpointNotificationKey(cfg.GetNetworkService(), label.GetKey(), label.GetValue())
			err := d.notifPublisher.Put(key, &empty.Empty{}, datasync.WithClientLifetimeTTL())
			if err != nil {
				d.log.Warnf("Failed to publish notification about created endpoint %+v: %v", cfg, err)
			}
		}
	}
	return metadata, nil
}

// Delete removes NSM endpoint.
func (d *NSMEndpointDescriptor) Delete(key string, cfg *nsm.NetworkServiceEndpoint, metadata interface{}) error {
	err := d.epDemux.DeleteEndpoint(context.Background(), cfg.GetNetworkService())
	if err != nil {
		d.log.Warn(err)
	}

	// publish notification about the removed endpoint
	if d.notifPublisher != nil {
		for _, label := range cfg.AdvertisedLabels {
			key := nsm.EndpointNotificationKey(cfg.GetNetworkService(), label.GetKey(), label.GetValue())
			_, err := d.notifPublisher.Delete(key)
			if err != nil {
				d.log.Warnf("Failed to publish notification about deleted endpoint %+v: %v", cfg, err)
			}
		}
	}
	return nil
}

// AddEndpoint registers new NSM endpoint with the demultiplexer.
func (x *NSMEndpointDemux) AddEndpoint(ctx context.Context, configuration *nsm_sdk_common.NSConfiguration,
	service networkservice.NetworkServiceServer, closer func() error) (err error) {
	x.Lock()
	defer x.Unlock()
	nseName := configuration.AdvertiseNseName

	// Create connection to NSM and start gRPC server if it has not been done already
	if !x.connected {
		// Connect to NSM.
		x.nsmConn, err = nsm_sdk_common.NewNSMConnection(ctx, configuration)
		if err != nil {
			x.log.Error(err)
			return err
		}

		// Start gRPC server for processing connection requests.
		x.grpcServer = nsm_sdk_tools.NewServer(ctx)
		networkservice.RegisterNetworkServiceServer(x.grpcServer, x)
		err = nsm_sdk_endpoint.Init(service, &nsm_sdk_endpoint.InitContext{
			GrpcServer: x.grpcServer,
		})
		if err != nil {
			return err
		}

		if err = nsm_sdk_tools.SocketCleanup(configuration.NsmClientSocket); err != nil {
			x.log.Errorf("Failed to cleanup stale socket %s with error: %v", configuration.NsmClientSocket, err)
			return err
		}
		connectionServer, err := net.Listen("unix", configuration.NsmClientSocket)
		if err != nil {
			x.log.Errorf("Failed to listen on a NSM socket %s with error: %v", configuration.NsmClientSocket, err)
			return err
		}
		x.log.Infof("Listening on NSM socket %s", configuration.NsmClientSocket)

		go func() {
			if err := x.grpcServer.Serve(connectionServer); err != nil {
				x.log.Fatalf("Failed to start grpc server on NSM socket %v with error: %v ",
					configuration.NsmClientSocket, err)
			}
		}()

		// Connect to NSM registry.
		x.nsmRegistry = nsm_registry.NewNetworkServiceRegistryClient(x.nsmConn.GrpcClient)
		x.connected = true
	}

	// Register new endpoint.
	span := nsm_span.FromContext(ctx, fmt.Sprintf("Endpoint-%v-Start", nseName))
	span.LogObject("labels", configuration.AdvertiseNseLabels)
	defer span.Finish()

	nse := &nsm_registry.NetworkServiceEndpoint{
		NetworkServiceName: nseName,
		Payload:            "IP",
		Labels:             nsm_sdk_tools.ParseKVStringToMap(configuration.AdvertiseNseLabels, ",", "="),
	}
	registration := &nsm_registry.NSERegistration{
		NetworkService: &nsm_registry.NetworkService{
			Name:    nseName,
			Payload: "IP",
		},
		NetworkServiceEndpoint: nse,
	}
	span.LogObject("nse-request", registration)

	registeredNSE, err := x.nsmRegistry.RegisterNSE(span.Context(), registration)
	if err != nil {
		x.log.Errorf("Unable to register NSM endpoint: %v", err)
		return err
	}
	endpointName := registeredNSE.GetNetworkServiceEndpoint().GetName()
	x.endpoints[nseName] = &endpointReg{
		EpName:  endpointName,
		Service: service,
		Close:   closer,
	}
	x.log.Infof("Added endpoint %s for network service %s", endpointName, nseName)
	span.LogObject("endpoint-name", endpointName)
	span.Logger().Infof("NSE registered: %v", registeredNSE)
	span.Logger().Infof("NSE: channel has been successfully advertised, waiting for connection from NSM...")
	return nil
}

// DeleteEndpoint unregisters NSM endpoint.
func (x *NSMEndpointDemux) DeleteEndpoint(ctx context.Context, nseName string) (err error) {
	x.Lock()
	defer x.Unlock()

	ep, epExists := x.endpoints[nseName]
	if !epExists {
		err = fmt.Errorf("no such endpoint: %s", nseName)
		x.log.Error(err)
		return err
	}

	// Close all connections with NSM clients first.
	err = ep.Close()
	if err != nil {
		x.log.Error(err)
		return err
	}

	// Unregister the endpoint.
	span := nsm_span.FromContext(ctx, fmt.Sprintf("Endpoint-%v-Delete", nseName))
	defer span.Finish()
	removeNSE := &nsm_registry.RemoveNSERequest{
		NetworkServiceEndpointName: ep.EpName,
	}
	span.LogObject("delete-request", removeNSE)
	_, err = x.nsmRegistry.RemoveNSE(span.Context(), removeNSE)
	if err != nil {
		span.Logger().Errorf("Failed removing NSE: %v, with %v", removeNSE, err)
		x.log.Error(err)
		return err
	}
	delete(x.endpoints, nseName)
	x.log.Infof("Deleted endpoint %s from network service %s", ep.EpName, nseName)
	return err
}

// Request routes the connection request to the corresponding endpoint by the network service name.
func (x *NSMEndpointDemux) Request(ctx context.Context,
	request *networkservice.NetworkServiceRequest) (*nsm_connection.Connection, error) {
	x.Lock()
	defer x.Unlock()

	nsmConn := request.GetConnection()
	nseName := nsmConn.GetNetworkService()
	ep, epExists := x.endpoints[nseName]
	if !epExists {
		err := fmt.Errorf("no such service: %v", nseName)
		x.log.Error(err)
		return nsmConn, err
	}
	x.log.Infof("Routing connection request (id=%s) to endpoint %s (nse: %s)",
		request.GetConnection().GetId(), ep.EpName, nseName)
	return ep.Service.Request(ctx, request)
}

// Request routes the request to close a connection to the corresponding endpoint by the network service name.
func (x *NSMEndpointDemux) Close(ctx context.Context, conn *nsm_connection.Connection) (*empty.Empty, error) {
	x.Lock()
	defer x.Unlock()

	nseName := conn.GetNetworkService()
	ep, epExists := x.endpoints[nseName]
	if !epExists {
		err := fmt.Errorf("no such service: %v", nseName)
		x.log.Error(err)
		return &empty.Empty{}, err
	}
	x.log.Infof("Routing request to close connection (id=%s) to endpoint %s (nse: %s)",
		conn.GetId(), ep.EpName, nseName)
	return ep.Service.Close(ctx, conn)
}

// Request creates the connection interface on the endpoint side.
func (e *NSMEndpoint) Request(ctx context.Context,
	request *networkservice.NetworkServiceRequest) (*nsm_connection.Connection, error) {

	// check if this is the same client trying to connect again
	nsmConn := request.GetConnection()
	if _, duplicate := e.connections[nsmConn.GetId()]; duplicate {
		e.log.Infof("Removing duplicate NSM connection=%+v", nsmConn)
		if err := e.closeConnection(ctx, nsmConn); err != nil {
			return nsmConn, err
		}
	}

	// check if more than one connection is allowed
	if e.cfg.GetSingleClient() && len(e.connections) > 0 {
		e.log.Info("[NSM Endpoint] - Cleaning up previous client connections")
		for connectionIndex := range e.connections {
			connection := e.connections[connectionIndex]
			e.log.Infof("[NSM Endpoint] - Removing duplicate NSM connection=%+v", connection)
			if err := e.closeConnection(ctx, connection.conn); err != nil {
				return nsmConn, err
			}
		}
		//return nsmConn, fmt.Errorf("only single client is expected to connect to this endpoint")
	}

	// create connection interface
	config, index, err := e.buildInterfaceConfig(e.cfg, nsmConn)
	if err != nil {
		e.log.Error(err)
		return nsmConn, err
	}
	c := nsmConnection{
		index: index,
		conn:  nsmConn,
	}
	e.connections[nsmConn.GetId()] = c
	_ = e.localclient.ChangeRequest().Update(config).Send(ctx)

	if nsm_sdk_endpoint.Next(ctx) != nil {
		return nsm_sdk_endpoint.Next(ctx).Request(ctx, request)
	}
	return nsmConn, nil
}

// Close removes the connection interface on the endpoint side.
func (e *NSMEndpoint) Close(ctx context.Context, nsmConn *nsm_connection.Connection) (*empty.Empty, error) {
	if err := e.closeConnection(ctx, nsmConn); err != nil {
		return &empty.Empty{}, err
	}

	if nsm_sdk_endpoint.Next(ctx) != nil {
		return nsm_sdk_endpoint.Next(ctx).Close(ctx, nsmConn)
	}
	return &empty.Empty{}, nil
}

// CloseConnections closes all active connections.
func (e *NSMEndpoint) CloseConnections() error {
	for _, c := range e.connections {
		err := e.closeConnection(context.Background(), c.conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *NSMEndpoint) closeConnection(ctx context.Context, nsmConn *nsm_connection.Connection) error {
	// remove the connection interface
	config, _, err := e.buildInterfaceConfig(e.cfg, nsmConn)
	if err != nil {
		e.log.Error(err)
		return err
	}
	txnCtx := context.Background()
	txnCtx = kvs.WithoutBlocking(txnCtx)
	_ = e.localclient.ChangeRequest().Delete(config).Send(txnCtx)

	// allow the connection index to be reused by future connections
	delete(e.connections, nsmConn.GetId())
	return nil
}

// Name returns the composite name.
func (e *NSMEndpoint) Name() string {
	return "NSM Plugin Endpoint"
}

// buildInterfaceConfig returns the configuration of the interface corresponding to the given connection.
func (e *NSMEndpoint) buildInterfaceConfig(cfg *nsm.NetworkServiceEndpoint, conn *nsm_connection.Connection) (
	iface proto.Message, index int, err error) {

	// TODO: consider using IPAM provided by NSM (see UniversalCNFVPPAgentBackend.ProcessEndpoint())

	interfaceName := cfg.GetInterfaceNamePrefix()
	if suffixFromLabel := cfg.GetInterfaceNameSuffixFromLabel(); suffixFromLabel != "" {
		suffix, hasLabel := conn.Labels[suffixFromLabel]
		if !hasLabel {
			err = fmt.Errorf("client does not define outgoing label '%s' used as interface name suffix",
				suffixFromLabel)
			return
		}
		interfaceName += suffix
	} else {
		index = e.getConnectionIndex(conn)
		interfaceName += strconv.Itoa(index)
	}

	switch cfg.GetInterfaceType() {
	case nsm.ConnectionInterfaceType_DEFAULT_INTERFACE:
		fallthrough
	case nsm.ConnectionInterfaceType_KERNEL_INTERFACE:
		kernelIface := &linux_interfaces.Interface{
			Name:       interfaceName,
			HostIfName: conn.GetMechanism().GetParameters()[nsm_common.InterfaceNameKey],
			Type:       linux_interfaces.Interface_EXISTING,
			Enabled:    true,
		}
		if cfg.GetSingleClient() {
			kernelIface.IpAddresses = cfg.GetIpAddresses()
			kernelIface.PhysAddress = cfg.GetPhysAddress()
		}
		iface = kernelIface
		return

	case nsm.ConnectionInterfaceType_MEM_INTERFACE:
		var baseDir string
		baseDir, err = nsmBaseDir()
		if err != nil {
			return
		}
		socketFilename := path.Join(baseDir, nsm_memif.ToMechanism(conn.GetMechanism()).GetSocketFilename())
		if err = os.MkdirAll(path.Dir(socketFilename), os.ModePerm); err != nil {
			return
		}
		memifIface := &vpp_interfaces.Interface{
			Name:    interfaceName,
			Type:    vpp_interfaces.Interface_MEMIF,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Memif{
				Memif: &vpp_interfaces.MemifLink{
					Master:         true, // The endpoint is always the master in MEMIF
					SocketFilename: socketFilename,
				},
			},
		}
		if cfg.GetSingleClient() {
			memifIface.IpAddresses = cfg.GetIpAddresses()
			memifIface.PhysAddress = cfg.GetPhysAddress()
		}
		iface = memifIface
		return
	}
	err = fmt.Errorf("unsupported interface type: %s", cfg.GetInterfaceType())
	return
}

func (e *NSMEndpoint) getConnectionIndex(conn *nsm_connection.Connection) int {
	c, has := e.connections[conn.GetId()]
	if !has {
		index := e.allocateConnectionIndex()
		e.log.Infof("Allocated index=%d for connection=%+v", index, conn)
		return index
	}
	return c.index
}

func (e *NSMEndpoint) allocateConnectionIndex() int {
	var index int
	for e.usesConnectionIndex(index) {
		index++
	}
	return index
}

func (e *NSMEndpoint) usesConnectionIndex(index int) bool {
	for _, conn := range e.connections {
		if index == conn.index {
			return true
		}
	}
	return false
}

func generateDummySubnet(connectionName string) string {
	h := fnv.New32a()
	h.Write([]byte(connectionName))
	hash := h.Sum32()
	return fmt.Sprintf("0.%d.%d.0/24", byte(hash%256), byte((hash>>8)%256))
}
