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
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/datasync/kvdbsync/local"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval/etcd"
	"github.com/ligato/cn-infra/health/probe"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/infra"
	"github.com/ligato/cn-infra/logging/logmanager"

	"go.cdnf.io/cnf-nsm/plugins/nsmplugin"
	vppagent "go.ligato.io/vpp-agent/v3/cmd/vpp-agent/app"
	"go.ligato.io/vpp-agent/v3/plugins/netalloc"
	"go.ligato.io/vpp-agent/v3/plugins/orchestrator"
)

// NSMAgent defines plugins which will be loaded and their order.
type NSMAgent struct {
	infra.PluginName
	LogManager *logmanager.Plugin

	vppagent.Linux
	Netalloc *netalloc.Plugin

	NSMPlugin *nsmplugin.NSMPlugin

	Orchestrator *orchestrator.Plugin
	ETCDDataSync *kvdbsync.Plugin

	Probe *probe.Plugin
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

	linux := vppagent.DefaultLinux()

	return &NSMAgent{
		PluginName:   "NSM-Agent",
		LogManager:   &logmanager.DefaultPlugin,
		Linux:        linux,
		Netalloc:     &netalloc.DefaultPlugin,
		NSMPlugin:    &nsmplugin.DefaultPlugin,
		Orchestrator: &orchestrator.DefaultPlugin,
		ETCDDataSync: etcdDataSync,
		Probe:        &probe.DefaultPlugin,
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
