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

package cnfconfig

import (
	"errors"
	"github.com/ligato/cn-infra/logging"

	contiv_customconfig "github.com/contiv/vpp/plugins/crd/handler/customconfiguration"
	contiv_kvdbreflector "github.com/contiv/vpp/plugins/crd/handler/kvdbreflector"
	contiv_crd "github.com/contiv/vpp/plugins/crd/pkg/apis/contivppio/v1"
	"go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/apis/pantheontech/v1"
	crdClientSet "go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/client/clientset/versioned"
)

// Handler implements the Handler interface for CRD<->KVDB Reflector, largely based on CustomConfiguration
// CRD from Contiv/VPP.
type Handler struct {
	Log       logging.Logger
	CrdClient *crdClientSet.Clientset
}

// CrdName returns name of the CRD.
func (h *Handler) CrdName() string {
	return "CNFConfiguration"
}

// CrdKeyPrefix returns the longest-common prefix under which the instances
// of the given CRD are reflected into KVDB.
func (h *Handler) CrdKeyPrefix() (prefix string, underKsrPrefix bool) {
	return "/vnf-agent/", false
}

// IsCrdKeySuffix always returns true - the key prefix does not overlap with
// other CRDs
func (h *Handler) IsCrdKeySuffix(keySuffix string) bool {
	return true
}

// CrdObjectToKVData converts the K8s representation of CNFConfiguration into the
// corresponding configuration for the destination CNF(s).
func (h *Handler) CrdObjectToKVData(obj interface{}) (data []contiv_kvdbreflector.KVData, err error) {
	cnfConfig, ok := obj.(*v1.CNFConfiguration)
	if !ok {
		return nil, errors.New("failed to cast into CNFConfiguration struct")
	}

	// do not reinvent the wheel, instead re-use CustomConfiguration CRD handler from Contiv/VPP
	contivHandler := &contiv_customconfig.Handler{Log: h.Log}
	return contivHandler.CrdObjectToKVData(
		&contiv_crd.CustomConfiguration{
			TypeMeta:   cnfConfig.TypeMeta,
			ObjectMeta: cnfConfig.ObjectMeta,
			Spec: contiv_crd.CustomConfigurationSpec{
				Microservice: cnfConfig.Spec.Microservice,
				ConfigItems:  cnfConfig.Spec.ConfigItems,
			},
			Status: cnfConfig.Status,
		})
}

// IsExclusiveKVDB returns true - configuration is passed to CNFs only through this CRD (at least for now)
func (h *Handler) IsExclusiveKVDB() bool {
	return true
}

// PublishCrdStatus updates the resource Status information.
func (h *Handler) PublishCrdStatus(obj interface{}, opRetval error) error {
	cnfConfig, ok := obj.(*v1.CNFConfiguration)
	if !ok {
		return errors.New("failed to cast into CNFConfiguration struct")
	}
	cnfConfig = cnfConfig.DeepCopy()
	if opRetval == nil {
		cnfConfig.Status.Status = v1.StatusSuccess
	} else {
		cnfConfig.Status.Status = v1.StatusFailure
		cnfConfig.Status.Message = opRetval.Error()
	}
	_, err := h.CrdClient.PantheonV1().CNFConfigurations(cnfConfig.Namespace).Update(cnfConfig)
	return err
}
