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
	"os"
	"path"

	"github.com/golang/protobuf/proto"

	"go.ligato.io/cn-infra/v2/logging"
	"go.ligato.io/vpp-agent/v3/client"
	kvs "go.ligato.io/vpp-agent/v3/plugins/kvscheduler/api"
	linux_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/linux/interfaces"
	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	"go.cdnf.io/cnf-nsm/plugins/nsmplugin/descriptor/adapter"
	"go.cdnf.io/cnf-nsm/plugins/nsmplugin/nsmetadata"
	"go.cdnf.io/cnf-nsm/proto/nsm"

	nsm_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	nsm_kernel "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	nsm_memif "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	nsm_sdk_client "github.com/networkservicemesh/networkservicemesh/sdk/client"
	nsm_sdk_common "github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	nsmClientDescriptorName = "nsm-client-descriptor"
	nsmInvalidMechanims     = "<invalid>"
)

// validation errors
var (
	// ErrClientWithoutName is returned when NSM client is configured with undefined name.
	ErrClientWithoutName = errors.New("NSM client defined without logical name")

	// ErrClientWithoutNetworkService is returned when NSM client is configured with undefined network service name.
	ErrClientWithoutNetworkService = errors.New("NSM client defined without network service name")

	// ErrClientWithoutInterface is returned when NSM client is configured with undefined interface name.
	ErrClientWithoutInterface = errors.New("NSM client defined without interface name")
)

// NSMClientDescriptor creates client connections to NSM endpoints.
type NSMClientDescriptor struct {
	// dependencies
	log         logging.Logger
	localclient client.ConfigClient
}

// NewNSMClientDescriptor creates new instance of the descriptor.
func NewNSMClientDescriptor(log logging.PluginLogger, localclient client.ConfigClient) (descrCtx *NSMClientDescriptor, descr *kvs.KVDescriptor) {
	descrCtx = &NSMClientDescriptor{
		log:         log.NewLogger(nsmClientDescriptorName),
		localclient: localclient,
	}
	typedDescr := &adapter.NSMClientDescriptor{
		Name:          nsmClientDescriptorName,
		KeySelector:   nsm.ModelNSMClient.IsKeyValid,
		ValueTypeName: nsm.ModelNSMClient.ProtoName(),
		KeyLabel:      nsm.ModelNSMClient.StripKeyPrefix,
		NBKeyPrefix:   nsm.ModelNSMClient.KeyPrefix(),
		WithMetadata:  true,
		Validate:      descrCtx.Validate,
		Create:        descrCtx.Create,
		Delete:        descrCtx.Delete,
	}
	return descrCtx, adapter.NewNSMClientDescriptor(typedDescr)
}

// Validate validates NSM-Client configuration.
func (d *NSMClientDescriptor) Validate(key string, cfg *nsm.NetworkServiceClient) error {
	// validate name
	if cfg.GetName() == "" {
		return kvs.NewInvalidValueError(ErrClientWithoutName, "name")
	}
	if cfg.GetNetworkService() == "" {
		return kvs.NewInvalidValueError(ErrClientWithoutNetworkService, "network_service")
	}
	if cfg.GetInterfaceName() == "" {
		return kvs.NewInvalidValueError(ErrClientWithoutInterface, "interface_name")
	}
	return nil
}

// Creates establishes connection with NSM endpoint and creates the associated interface.
func (d *NSMClientDescriptor) Create(key string, cfg *nsm.NetworkServiceClient) (metadata *nsmetadata.NsmClientMetadata, err error) {
	// initialize NSM client
	configuration := nsm_sdk_common.FromEnv()
	configuration.MechanismType = nsmMechanismFrominterfaceType(cfg.GetInterfaceType())
	configuration.OutgoingNscName = cfg.GetNetworkService()
	configuration.OutgoingNscLabels = nsmlabelsFromProto(cfg.GetOutgoingLabels())

	// Currently NSM is unable/refuses to create IP-less connections (SrcIpRequired & DstIpRequired are always enabled
	// by NsmClient.Connect). Therefore we generate a dummy subnet from within the reserved IP address block 0.0.0.0/8
	// that is unlikely to collide with subnets generated for other clients/endpoints of this agent instance.
	configuration.IPAddress = generateDummySubnet("client-" + cfg.GetName())

	nsmClient, err := nsm_sdk_client.NewNSMClient(context.Background(), configuration)
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	// try to create connection with the network service
	nsmConn, err := nsmClient.Connect(context.Background(), cfg.GetInterfaceName(),
		configuration.MechanismType, cfg.GetName())
	if err != nil {
		_ = nsmClient.Destroy(context.Background())
		d.log.Error(err)
		return nil, err
	}

	// create connection interface
	config, err := d.buildInterfaceConfig(cfg, nsmConn)
	if err != nil {
		_ = nsmClient.Close(context.Background(), nsmConn)
		_ = nsmClient.Destroy(context.Background())
		d.log.Error(err)
		return nil, err
	}
	go func() {
		ctx := context.Background()
		// TODO: non-blocking transactions are currently broken in the agent on the orchestrator side
		//ctx = kvs.WithoutBlocking(ctx)
		_ = d.localclient.ChangeRequest().Update(config...).Send(ctx)
	}()

	metadata = &nsmetadata.NsmClientMetadata{
		NsmClient: nsmClient,
		NsmConn:   nsmConn,
	}
	return metadata, nil
}

// Delete removes interfaces and closes connection with NSM endpoint.
func (d *NSMClientDescriptor) Delete(key string, cfg *nsm.NetworkServiceClient, metadata *nsmetadata.NsmClientMetadata) error {
	// remove connection interface
	config, err := d.buildInterfaceConfig(cfg, metadata.NsmConn)
	if err != nil {
		d.log.Error(err)
		return err
	}
	go func() {
		ctx := context.Background()
		// TODO: non-blocking transactions are currently broken in the agent on the orchestrator side
		//ctx = kvs.WithoutBlocking(ctx)
		_ = d.localclient.ChangeRequest().Delete(config...).Send(ctx)
	}()

	// close the connection
	err = metadata.NsmClient.Close(context.Background(), metadata.NsmConn)
	if err != nil {
		d.log.Error(err)
		return err
	}

	// destroy the client
	err = metadata.NsmClient.Destroy(context.Background())
	if err != nil {
		d.log.Error(err)
		return err
	}
	return nil
}

// buildInterfaceConfig returns the configuration of the interface corresponding to the given connection.
func (d *NSMClientDescriptor) buildInterfaceConfig(cfg *nsm.NetworkServiceClient, conn *nsm_connection.Connection) (
	[]proto.Message, error) {

	// TODO: consider using IPAM provided by NSM (see UniversalCNFVPPAgentBackend.ProcessClient())

	switch cfg.GetInterfaceType() {
	case nsm.ConnectionInterfaceType_DEFAULT_INTERFACE:
		fallthrough
	case nsm.ConnectionInterfaceType_KERNEL_INTERFACE:
		iface := &linux_interfaces.Interface{
			Name:        cfg.GetInterfaceName(),
			Type:        linux_interfaces.Interface_EXISTING,
			Enabled:     true,
			IpAddresses: cfg.GetIpAddresses(),
			PhysAddress: cfg.GetPhysAddress(),
		}
		return []proto.Message{iface}, nil

	case nsm.ConnectionInterfaceType_MEM_INTERFACE:
		baseDir, err := nsmBaseDir()
		if err != nil {
			return nil, err
		}
		socketFilename := path.Join(baseDir, nsm_memif.ToMechanism(conn.GetMechanism()).GetSocketFilename())
		iface := &vpp_interfaces.Interface{
			Name:        cfg.GetInterfaceName(),
			Type:        vpp_interfaces.Interface_MEMIF,
			Enabled:     true,
			IpAddresses: cfg.GetIpAddresses(),
			PhysAddress: cfg.GetPhysAddress(),
			Link: &vpp_interfaces.Interface_Memif{
				Memif: &vpp_interfaces.MemifLink{
					Master:         false, // The client is not the master in MEMIF
					SocketFilename: socketFilename,
				},
			},
		}
		return []proto.Message{iface}, nil
	}
	return nil, fmt.Errorf("unsupported interface type: %s", cfg.GetInterfaceType())
}

func nsmMechanismFrominterfaceType(interfaceType nsm.ConnectionInterfaceType) string {
	switch interfaceType {
	case nsm.ConnectionInterfaceType_DEFAULT_INTERFACE:
		fallthrough
	case nsm.ConnectionInterfaceType_KERNEL_INTERFACE:
		return nsm_kernel.MECHANISM
	case nsm.ConnectionInterfaceType_MEM_INTERFACE:
		return nsm_memif.MECHANISM
	}
	return nsmInvalidMechanims
}

func nsmBaseDir() (string, error) {
	baseDir, ok := os.LookupEnv(nsm_sdk_common.WorkspaceEnv)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not defined", nsm_sdk_common.WorkspaceEnv)
	}
	return baseDir, nil
}

func nsmlabelsFromProto(labels []*nsm.ConnectionLabel) string {
	labelString := ""
	for _, label := range labels {
		if len(labelString) > 0 {
			labelString += ","
		}
		labelString += label.Key + "=" + label.Value
	}
	return labelString
}
