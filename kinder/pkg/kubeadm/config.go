/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubeadm

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
)

// Config returns a kubeadm config generated using the config API version
// and with the customizable settings based on data
func Config(kubeadmConfigVersion string, data ConfigData) (config string, err error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing kubeadm config %s", kubeadmConfigVersion)
	var templateSource string
	switch kubeadmConfigVersion {
	case "v1beta2":
		templateSource = configTemplateBetaV2
	case "v1beta3":
		templateSource = configTemplateBetaV3
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	t, err := template.New("kubeadm-config").Parse(templateSource)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// derive any automatic fields if not supplied
	data.Derive()

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}

	return buff.String(), nil
}

// GetKubeadmConfigVersion returns the kubeadm config version corresponding to a Kubernetes kubeadmVersion
func GetKubeadmConfigVersion(kubeadmVersion *K8sVersion.Version) string {
	// v1alpha1 (that is Kubernetes v1.10.0) is out of support
	// v1alpha2 (that is Kubernetes v1.11.0) is out of support
	// v1alpha3 (that is Kubernetes v1.13.0) is out of support
	if kubeadmVersion.Major() > 1 || kubeadmVersion.Minor() >= 22 {
		return "v1beta3"
	}
	return "v1beta2"
}

// ConfigData is supplied to the kubeadm config template, with values populated
// by the cluster package
type ConfigData struct {
	ClusterName       string
	KubernetesVersion string
	// The ControlPlaneEndpoint, that is the address of the external loadbalancer
	// if defined or the bootstrap node
	ControlPlaneEndpoint string
	// The Local API Server port
	APIBindPort int
	// The API server external listen IP (which we will port forward)
	APIServerAddress string
	// ControlPlane flag specifies the node belongs to the control plane
	ControlPlane bool
	// The main IP address of the node
	NodeAddress string
	// The Token for TLS bootstrap
	Token string
	// The subnet used for pods
	PodSubnet string
	// The subnet used for services
	ServiceSubnet string
	// IPv4 values take precedence over IPv6 by default, if true set IPv6 default values
	IPv6 bool
	// The kubeadm feature-gate
	FeatureGateName  string
	FeatureGateValue string
	// DerivedConfigData is populated by Derive()
	// These auto-generated fields are available to Config templates,
	// but not meant to be set by hand
	DerivedConfigData
}

// DerivedConfigData fields are automatically derived by
// ConfigData.Derive if they are not specified / zero valued
type DerivedConfigData struct {
	// DockerStableTag is automatically derived from KubernetesVersion
	DockerStableTag string
}

// Derive automatically derives DockerStableTag if not specified
func (c *ConfigData) Derive() {
	if c.DockerStableTag == "" {
		c.DockerStableTag = strings.Replace(c.KubernetesVersion, "+", "_", -1)
	}
}

// See docs for these APIs at:
// https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm#section-directories

// configTemplateBetaV2 is the kubeadm config template for API version v1beta2
const configTemplateBetaV2 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
controlPlaneEndpoint: "{{ .ControlPlaneEndpoint }}"
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost, "{{.APIServerAddress}}"]
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::"
    {{- end }}
scheduler:
  extraArgs:
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    address: "::"
    bind-address: "::1"
    {{- end }}
networking:
  podSubnet: "{{ .PodSubnet }}"
  serviceSubnet: "{{ .ServiceSubnet }}"
{{ if .FeatureGateName -}}
featureGates:
  {{ .FeatureGateName }}: {{ .FeatureGateValue }}
{{- end }}
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network.
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  advertiseAddress: "{{ .NodeAddress }}"
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
{{ if .ControlPlane -}}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "{{ .NodeAddress }}"
    bindPort: {{.APIBindPort}}
{{- end }}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
discovery:
  bootstrapToken:
    apiServerEndpoint: "{{ .ControlPlaneEndpoint }}"
    token: "{{ .Token }}"
    unsafeSkipCAVerification: true
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
# configure ipv6 addresses in IPv6 mode
{{ if .IPv6 -}}
address: "::"
healthzBindAddress: "::"
{{- end }}
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
failSwapOn: false
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
# pin the cgroup driver to systemd.
# this assumes that the CR on the node image is configured accordingly.
cgroupDriver: "systemd"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`

// configTemplateBetaV3 is the kubeadm config template for API version v1beta3
const configTemplateBetaV3 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
controlPlaneEndpoint: "{{ .ControlPlaneEndpoint }}"
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost, "{{.APIServerAddress}}"]
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::"
    {{- end }}
scheduler:
  extraArgs:
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    address: "::"
    bind-address: "::1"
    {{- end }}
networking:
  podSubnet: "{{ .PodSubnet }}"
  serviceSubnet: "{{ .ServiceSubnet }}"
{{ if .FeatureGateName -}}
featureGates:
  {{ .FeatureGateName }}: {{ .FeatureGateValue }}
{{- end }}
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network.
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  advertiseAddress: "{{ .NodeAddress }}"
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
metadata:
  name: config
{{ if .ControlPlane -}}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "{{ .NodeAddress }}"
    bindPort: {{.APIBindPort}}
{{- end }}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
discovery:
  bootstrapToken:
    apiServerEndpoint: "{{ .ControlPlaneEndpoint }}"
    token: "{{ .Token }}"
    unsafeSkipCAVerification: true
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
# configure ipv6 addresses in IPv6 mode
{{ if .IPv6 -}}
address: "::"
healthzBindAddress: "::"
{{- end }}
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
failSwapOn: false
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
# pin the cgroup driver to systemd.
# this assumes that the CR on the node image is configured accordingly.
cgroupDriver: "systemd"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`
