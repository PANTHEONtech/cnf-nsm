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

package nsmetadata

import (
	nsm_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	nsm_client "github.com/networkservicemesh/networkservicemesh/sdk/client"
	nsm_endpoint "github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// NsmClientMetadata encapsulates metadata stored alongside opened client connection.
type NsmClientMetadata struct {
	NsmClient *nsm_client.NsmClient
	NsmConn   *nsm_connection.Connection
}

// Endpoint is implemented by NSMPluginEndpoint.
type Endpoint interface {
	CloseConnections() error
}

// NsmEndpointMetadata encapsulates metadata stored alongside running endpoint.
type NsmEndpointMetadata struct {
	Endpoint    Endpoint
	NsmEndpoint nsm_endpoint.NsmEndpoint
}
