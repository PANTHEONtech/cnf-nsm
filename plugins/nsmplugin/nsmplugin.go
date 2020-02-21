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

package nsmplugin

import (
	"github.com/ligato/cn-infra/infra"

	"go.cdnf.io/cnf-nsm/plugins/nsmplugin/descriptor"
	"go.ligato.io/vpp-agent/v3/client"
	kvs "go.ligato.io/vpp-agent/v3/plugins/kvscheduler/api"
)

//go:generate descriptor-adapter --descriptor-name NSMClient --value-type *nsm.NetworkServiceClient --meta-type *nsmetadata.NsmClientMetadata --import "go.cdnf.io/cnf-nsm/proto/nsm" --import "go.cdnf.io/cnf-nsm/plugins/nsmplugin/nsmetadata" --output-dir "descriptor"
//go:generate descriptor-adapter --descriptor-name NSMEndpoint --value-type *nsm.NetworkServiceEndpoint --meta-type *nsmetadata.NsmEndpointMetadata --import "go.cdnf.io/cnf-nsm/proto/nsm" --import "go.cdnf.io/cnf-nsm/plugins/nsmplugin/nsmetadata" --output-dir "descriptor"

// NSMPlugin allows integration with Network Service Mesh.
type NSMPlugin struct {
	Deps

	nsmClientDescriptor   *descriptor.NSMClientDescriptor
	nsmEndpointDescriptor *descriptor.NSMEndpointDescriptor
}

type Deps struct {
	infra.PluginDeps
	KVScheduler    kvs.KVScheduler
	NotifPublisher descriptor.NotificationPublisher // optional
}

// Init initializes NSM descriptors.
func (p *NSMPlugin) Init() error {
	// init & register descriptors
	var kvDescriptor *kvs.KVDescriptor
	p.nsmClientDescriptor, kvDescriptor = descriptor.NewNSMClientDescriptor(p.Log, client.LocalClient)
	err := p.KVScheduler.RegisterKVDescriptor(kvDescriptor)
	if err != nil {
		return err
	}
	p.nsmEndpointDescriptor, kvDescriptor = descriptor.NewNSMEndpointDescriptor(p.Log, client.LocalClient, p.NotifPublisher)
	err = p.KVScheduler.RegisterKVDescriptor(kvDescriptor)
	if err != nil {
		return err
	}
	return nil
}

// Close is NOOP.
func (p *NSMPlugin) Close() error {
	return nil
}
