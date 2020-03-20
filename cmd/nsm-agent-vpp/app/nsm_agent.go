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

package app

import (
	"go.ligato.io/cn-infra/v2/datasync"
	"go.ligato.io/cn-infra/v2/datasync/kvdbsync"
	"go.ligato.io/cn-infra/v2/datasync/kvdbsync/local"
	"go.ligato.io/cn-infra/v2/datasync/resync"
	"go.ligato.io/cn-infra/v2/db/keyval/etcd"
	"go.ligato.io/cn-infra/v2/health/probe"
	"go.ligato.io/cn-infra/v2/health/statuscheck"
	"go.ligato.io/cn-infra/v2/infra"
	"go.ligato.io/cn-infra/v2/logging/logmanager"

	"go.cdnf.io/cnf-nsm/plugins/nsmplugin"
	vppagent "go.ligato.io/vpp-agent/v3/cmd/vpp-agent/app"
	"go.ligato.io/vpp-agent/v3/plugins/configurator"
	linux_ifplugin "go.ligato.io/vpp-agent/v3/plugins/linux/ifplugin"
	linux_nsplugin "go.ligato.io/vpp-agent/v3/plugins/linux/nsplugin"
	"go.ligato.io/vpp-agent/v3/plugins/netalloc"
	"go.ligato.io/vpp-agent/v3/plugins/orchestrator"
	"go.ligato.io/vpp-agent/v3/plugins/restapi"
	"go.ligato.io/vpp-agent/v3/plugins/telemetry"
	vpp_ifplugin "go.ligato.io/vpp-agent/v3/plugins/vpp/ifplugin"
)

// NSMAgent defines plugins which will be loaded and their order.
type NSMAgent struct {
	infra.PluginName
	LogManager *logmanager.Plugin

	vppagent.VPP
	vppagent.Linux
	Netalloc *netalloc.Plugin

	NSMPlugin *nsmplugin.NSMPlugin

	Orchestrator *orchestrator.Plugin
	ETCDDataSync *kvdbsync.Plugin

	Configurator *configurator.Plugin
	RESTAPI      *restapi.Plugin
	Probe        *probe.Plugin
	Telemetry    *telemetry.Plugin
}

// NewAgent creates a new NSMAgent instance
func NewAgent() *NSMAgent {
	etcdDataSync := kvdbsync.NewPlugin(kvdbsync.UseKV(&etcd.DefaultPlugin))

	writers := datasync.KVProtoWriters{etcdDataSync}
	statuscheck.DefaultPlugin.Transport = writers

	watchers := datasync.KVProtoWatchers{
		local.DefaultRegistry,
		etcdDataSync,
	}
	orchestrator.DefaultPlugin.Watcher = watchers

	vpp_ifplugin.DefaultPlugin.LinuxIfPlugin = &linux_ifplugin.DefaultPlugin
	vpp_ifplugin.DefaultPlugin.NsPlugin = &linux_nsplugin.DefaultPlugin
	linux_ifplugin.DefaultPlugin.VppIfPlugin = &vpp_ifplugin.DefaultPlugin

	vpp := vppagent.DefaultVPP()
	linux := vppagent.DefaultLinux()

	return &NSMAgent{
		PluginName:   "NSM-Agent",
		LogManager:   &logmanager.DefaultPlugin,
		VPP:          vpp,
		Linux:        linux,
		Netalloc:     &netalloc.DefaultPlugin,
		NSMPlugin:    &nsmplugin.DefaultPlugin,
		Orchestrator: &orchestrator.DefaultPlugin,
		ETCDDataSync: etcdDataSync,
		Configurator: &configurator.DefaultPlugin,
		RESTAPI:      &restapi.DefaultPlugin,
		Probe:        &probe.DefaultPlugin,
		Telemetry:    &telemetry.DefaultPlugin,
	}
}

// Init initializes NSM agent.
func (a *NSMAgent) Init() error {
	return nil
}

// AfterInit triggers startup resync.
func (a *NSMAgent) AfterInit() error {
	resync.DefaultPlugin.DoResync()
	return nil
}

// Close closes NSM agent.
func (a *NSMAgent) Close() error {
	return nil
}
