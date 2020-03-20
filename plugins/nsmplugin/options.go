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
	"go.ligato.io/cn-infra/v2/config"
	"go.ligato.io/cn-infra/v2/datasync/kvdbsync"
	"go.ligato.io/cn-infra/v2/db/keyval"
	"go.ligato.io/cn-infra/v2/db/keyval/etcd"
	"go.ligato.io/cn-infra/v2/logging"
	"go.ligato.io/cn-infra/v2/servicelabel"

	"go.ligato.io/vpp-agent/v3/plugins/kvscheduler"

	"go.cdnf.io/cnf-nsm/proto/nsm"
)

const (
	// PluginName defines name of the NSM Plugin.
	// Config file name is `PluginName + ".conf"`
	PluginName = "nsm-plugin"
)

// DefaultPlugin is a default instance of NSMPlugin.
// Notifications enabled by default for the sake of CNF Provisioner.
var DefaultPlugin = *NewPlugin(WithNotifications())

// NewPlugin creates a new Plugin with the provides Options
func NewPlugin(opts ...Option) *NSMPlugin {
	p := &NSMPlugin{}
	p.PluginName = PluginName
	p.KVScheduler = &kvscheduler.DefaultPlugin

	for _, o := range opts {
		o(p)
	}

	if p.Log == nil {
		p.Log = logging.ForPlugin(p.String())
	}

	if p.Cfg == nil {
		p.Cfg = config.ForPlugin(p.String())
	}

	return p
}

// Option is a function that can be used in NewPlugin to customize Plugin.
type Option func(*NSMPlugin)

// WithNotifications enables notifications with default parameters (etcd DB, global group).
func WithNotifications() Option {
	return WithCustomizedNotifications(&etcd.DefaultPlugin, nsm.GlobalNotifAgentLabel)
}

// WithNotifications enables notifications with customized parameters.
func WithCustomizedNotifications(kvdb keyval.KvProtoPlugin, groupName string) Option {
	return func(p *NSMPlugin) {
		p.NotifPublisher = kvdbsync.NewPlugin(kvdbsync.UseDeps(
			func(p *kvdbsync.Deps) {
				p.PluginName = "nsm-notif-kvdb"
				p.KvPlugin = kvdb
				p.ServiceLabel = servicelabel.NewPlugin(ServiceLabelForNSMNotifs(groupName))
			}))
	}
}

func ServiceLabelForNSMNotifs(groupName string) servicelabel.Option {
	return func(p *servicelabel.Plugin) {
		p.PluginName = "nsm-notif-service-label"
		p.MicroserviceLabel = groupName
	}
}
