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

package nsm

import (
	"strings"

	"go.ligato.io/vpp-agent/v3/pkg/models"
)

const ModuleName = "cnf.nsm"

var ModelNSMClient = models.Register(
	&NetworkServiceClient{},
	models.Spec{
		Module:  ModuleName,
		Version: "v1",
		Type:    "client",
	},
)

var ModelNSMEndpoint = models.Register(
	&NetworkServiceEndpoint{},
	models.Spec{
		Module:  ModuleName,
		Version: "v1",
		Type:    "endpoint",
	},
	models.WithNameTemplate(
		"{{.NetworkService}}",
	),
)

// notifications
const (
	// GlobalNotifAgentLabel is an agent label used by default by all the agents with the NSM plugin to publish notifications.
	// In an unlikely case of sharing the same KVDB across multiple K8s clusters, use different label for each cluster
	// (to prevent from collisions).
	GlobalNotifAgentLabel = "global"

	// NotificationKeyPrefix is a prefix for keys of all notifications published by the NSM plugin via KVDB.
	NotificationKeyPrefix = "notification/nsm/"

	// EndpointNotificationKeyPrefix is a prefix for keys of all endpoint-related notifications published by the NSM
	// plugin via KVDB.
	EndpointNotificationKeyPrefix = NotificationKeyPrefix + "endpoint/"

	endpointNotifKeyTemplate = EndpointNotificationKeyPrefix + "service/{network-service}/label/{label-key}/{label-value}"
)

const (
	// InvalidKeyPart is used in key for parts which are invalid
	InvalidKeyPart = "<invalid>"
)

func ClientKey() string {
	return models.Key(&NetworkServiceClient{})
}

func EndpointKey() string {
	return models.Key(&NetworkServiceEndpoint{})
}

func EndpointNotificationKey(ns, labelKey, labelValue string) string {
	if ns == "" {
		ns = InvalidKeyPart
	}
	if labelKey == "" {
		labelKey = InvalidKeyPart
	}
	if labelValue == "" {
		labelValue = InvalidKeyPart
	}

	key := strings.Replace(endpointNotifKeyTemplate, "{network-service}", ns, 1)
	key = strings.Replace(key, "{label-key}", labelKey, 1)
	key = strings.Replace(key, "{label-value}", labelValue, 1)
	return key
}

// ParseEndpointNotificationKey parses network service name and the endpoint advertised label from a key
// used for a notification about a created endpoint.
func ParseEndpointNotificationKey(key string) (ns, labelKey, labelValue string, ok bool) {
	suffix := strings.TrimPrefix(key, EndpointNotificationKeyPrefix)
	if suffix == key {
		return
	}
	parts := strings.Split(key, "/")
	labelIdx := len(parts) - 3
	if len(parts) < 5 || parts[0] != "service" || parts[labelIdx] != "label" {
		return
	}

	ok = true
	ns = strings.Join(parts[2:labelIdx], "/")
	labelKey = parts[labelIdx+1]
	labelValue = parts[labelIdx+2]
	return
}
