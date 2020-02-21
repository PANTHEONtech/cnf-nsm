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

package crdplugin

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/namsral/flag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/db/keyval/etcd"
	"github.com/ligato/cn-infra/infra"
	"github.com/ligato/cn-infra/logging"

	contiv_controller "github.com/contiv/vpp/plugins/crd/controller"
	contiv_customconfig "github.com/contiv/vpp/plugins/crd/handler/customconfiguration"
	contiv_kvdbreflector "github.com/contiv/vpp/plugins/crd/handler/kvdbreflector"

	"go.cdnf.io/cnf-nsm/plugins/crdplugin/cnfconfig"
	"go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/apis/pantheontech"
	"go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/apis/pantheontech/v1"

	crdClientSet "go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/client/clientset/versioned"
	factory "go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/client/informers/externalversions"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

const (
	k8sResyncInterval = 10 * time.Minute
)

// Plugin implements CRDs used by Pantheon CNFs.
type Plugin struct {
	Deps
	verbose bool

	cnfConfigController *contiv_controller.CrdController

	ctx    context.Context
	cancel context.CancelFunc

	crdClient     *crdClientSet.Clientset
	apiclientset  *apiextcs.Clientset
	sharedFactory factory.SharedInformerFactory
}

// Deps defines dependencies of CRD plugin.
type Deps struct {
	infra.PluginDeps

	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig config.PluginConfig

	// etcd plugin is specifically required for raw-data support
	Etcd *etcd.Plugin
}

// Init initializes Kubernetes client.
func (p *Plugin) Init() error {
	var err error
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.Log.SetLevel(logging.DebugLevel)
	p.verbose = flag.Lookup("verbose").Value.String() == "true"

	kubeconfig := p.KubeConfig.GetConfigName()
	p.Log.WithField("kubeconfig", kubeconfig).Info("Loading kubernetes client config")
	k8sClientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client config: %s", err)
	}

	p.crdClient, err = crdClientSet.NewForConfig(k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build crd Client: %s", err)
	}

	p.apiclientset, err = apiextcs.NewForConfig(k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build api Client: %s", err)
	}

	p.sharedFactory = factory.NewSharedInformerFactory(p.crdClient, k8sResyncInterval)

	// init and start CRD controllers only after connection with etcd has been established
	p.Etcd.OnConnect(p.onEtcdConnect)
	return nil
}

// initializeCRDs initializes and starts controllers for all CRDs.
// Must be run *after* (first) connection with etcd has been established.
func (p *Plugin) initializeCRDs() error {
	cnfConfigInformer := p.sharedFactory.Pantheon().V1().CNFConfigurations().Informer()
	cnfConfigLog := p.Log.NewLogger("cnfConfigHandler")
	p.cnfConfigController = &contiv_controller.CrdController{
		Deps: contiv_controller.Deps{
			Log:       p.Log.NewLogger("cnfConfigController"),
			APIClient: p.apiclientset,
			Informer:  cnfConfigInformer,
			EventHandler: &contiv_kvdbreflector.KvdbReflector{
				Deps: contiv_kvdbreflector.Deps{
					Log:      cnfConfigLog,
					Publish:  p.Etcd.RawAccess(),
					Informer: cnfConfigInformer,
					Handler: &cnfconfig.Handler{
						Log:       cnfConfigLog,
						CrdClient: p.crdClient,
					},
				},
			},
		},
		Spec: contiv_controller.CrdSpec{
			TypeName:   reflect.TypeOf(v1.CNFConfiguration{}).Name(),
			Group:      pantheontech.GroupName,
			Version:    "v1",
			Plural:     "cnfconfigurations",
			Validation: contiv_customconfig.Validation(),
		},
	}

	if err := p.cnfConfigController.Init(); err != nil {
		return err
	}
	if p.verbose {
		p.cnfConfigController.Log.SetLevel(logging.DebugLevel)
		cnfConfigLog.SetLevel(logging.DebugLevel)
	}

	return nil
}

// onEtcdConnect is called when the connection with etcd is made.
// CRD controllers are initialized and started.
func (p *Plugin) onEtcdConnect() error {
	// Init and run the controllers
	err := p.initializeCRDs()
	if err != nil {
		return err
	}
	go func() {
		go p.cnfConfigController.Run(p.ctx.Done())
	}()
	return nil
}

// Close closes the plugin.
func (p *Plugin) Close() error {
	p.cancel()
	return nil
}
