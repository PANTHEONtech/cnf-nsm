### This repository is archived. No worries, we're still [here](https://pantheon.tech/contact-us/).

```
 █████  ██████   ██████ ██   ██ ██ ██    ██ ███████ ██████ 
██   ██ ██   ██ ██      ██   ██ ██ ██    ██ ██      ██   ██
███████ ██████  ██      ███████ ██ ██    ██ █████   ██   ██
██   ██ ██   ██ ██      ██   ██ ██  ██  ██  ██      ██   ██
██   ██ ██   ██  ██████ ██   ██ ██   ████   ███████ ██████                        
```
<br><br><br><br><br><br><br>

Seamless integration of CNFs with NSM
-------------------------------------

### NSM Intro

Recently, [Network Service Mesh (NSM)][nsm] has been drawing lot's of attention in the area
of network function virtualization (NFV). Inspired by Istio, Network Service Mesh
maps the concept of a Service Mesh to L2/L3 payloads. It runs on top of (any) CNI
and builds additional connections between Kubernetes Pods in the run-time based on Network Service
definition deployed via CRD. Unlike [Contiv-VPP][contiv-vpp], for example, NSM is mostly controlled
from within applications through provided SDK. This approach has its pros and cons. On one hand
it gives programmers more control over the interactions between their applications and
NSM, on the other hand it requires deeper understanding of the framework to get things
right. Another difference is that NSM intentionally offers only the minimalistic
point-to-point connections between pods (or clients and endpoints in their terminology)
and everything that can be implemented via CNFs is left out of the framework. Even things
as basic as connecting service chain with external physical interfaces of attaching multiple
services to a common L2/L3 network is not supported and instead left to the users (programmers)
of NSM to implement.

### Integration of NSM with Ligato

In Pantheon we see the potential of NSM and decided to tackle the main drawbacks
of the framework. For example, we have developed a [new plugin][nsm-plugin] for
[ligato-based][ligato-vpp-agent] control-plane agents that allows seamless integration
of CNFs with NSM. Instead of having to use the low-level and imperative NSM SDK,
the users (not necessary SW developers) can use the standard northbound (NB) protobuf
API to define the connections between their applications and other network services
in a declarative form. The plugin then uses NSM SDK behind the scenes to open
the connections and creates corresponding interfaces that the CNF is then ready to use.
The CNF components therefore do not have to care about how the interfaces were created,
whether it was by [Contiv][contiv-vpp], via NSM SDK, or in some other way, and can simply
use logical interface names for referencing. This approach allows to decouple
the implementation of the network function provided by a CNF from the service
networking/chaining that surrounds it.

The plugin for ligato-NSM integration is shipped both separately, ready for import into
existing ligato-based agents, and also as a part of our [NSM-Agent-VPP][nsm-agent-vpp]
and [NSM-Agent-Linux][nsm-agent-linux]. The former extends the vanilla [ligato VPP-Agent][ligato-vpp-agent]
with the NSM support while the latter also adds NSM support but omits all the VPP-related plugins
when only Linux networking needs to be managed.

Furthermore, since most of the common network features are already provided by 
[ligato VPP-Agent][ligato-vpp-agent], it is often unnecessary to do any additional
programming work whatsoever to develop a new CNF. With ligato framework and tools
developed at Pantheon, achieving a desired network function is often a matter of defining
network configuration in a declarative way inside one or more YAML files deployed as
Kubernetes CRD instances. For examples of ligato-based CNF deployments with NSM networking,
please refer to our [repository with CNF examples][cnf-examples].

Finally, included in the repository is also a [controller for K8s CRD][cnf-crd]
defined to allow deploying network configuration for ligato-based CNFs like any other
Kubernetes resource defined inside YAML-formatted files. Usage examples can also be
found in the repository with [CNF examples][cnf-examples].

[contiv-vpp]: https://github.com/contiv/vpp
[nsm]: https://networkservicemesh.io
[ligato-vpp-agent]: https://github.com/ligato/vpp-agent/
[crd]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[nsm-plugin]: plugins/nsmplugin/nsmplugin.go
[cnf-crd]:  cmd/cnf-crd/main.go
[nsm-agent-vpp]: cmd/nsm-agent-vpp/main.go
[nsm-agent-linux]: cmd/nsm-agent-linux/main.go
[cnf-examples]: https://github.com/PantheonTechnologies/cnf-examples
