# Testing pre-release versions of Kubernetes with kubeadm

## Overview

When testing Kubernetes with kubeadm three different version numbers should be considered:

- The version of .deb or .rpm packages, including kubelet, kubectl and related dependencies (e.g. kubernetes-cni);
  it is convenient to include also kubeadm package in this group to get the kubelet drop-in file deployed automatically.
- The version of the kubeadm binary to be used, that can eventually overwrite the kubeadm binary included in
  the .deb or .rpm packages. e.g. Use v1.10.3 GA packages for kubelet, kubectl, but test a local build of kubernetes.
- The `--kubernetes-version` flag of kubeadm or the `kubernetesVersion` field of the [kubeadm config file](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init),
  defining the version number of the control plane components installed by kubeadm.

For each of the above set version number, it is also necessary to understand the supported version skew policy,
the availability of pre-compiled artifacts, or, as alternative, how to create a local version through the build process.

Once all the required pieces will be available, it is possible to create a Kubernetes cluster with kubeadm and
execute tests.

<!-- TOC -->

- [Testing pre-release versions of Kubernetes with kubeadm](#testing-pre-release-versions-of-kubernetes-with-kubeadm)
    - [Overview](#overview)
    - [Kubeadm version skew policy](#kubeadm-version-skew-policy)
    - [Availability of pre-compiled release artifacts](#availability-of-pre-compiled-release-artifacts)
        - [Getting .deb or .rpm packages form repository](#getting-deb-or-rpm-packages-form-repository)
        - [Getting .deb or .rpm packages form a GCS bucket](#getting-deb-or-rpm-packages-form-a-gcs-bucket)
        - [Getting kubeadm binaries from a GCS bucket](#getting-kubeadm-binaries-from-a-gcs-bucket)
        - [Getting docker images from a GCR registry](#getting-docker-images-from-a-gcr-registry)
        - [Getting kubeadm binaries or docker images form github release page](#getting-kubeadm-binaries-or-docker-images-form-github-release-page)
    - [Create a local version](#create-a-local-version)
        - [Build .debs packages](#build-debs-packages)
        - [Build kubeadm binary](#build-kubeadm-binary)
        - [Build controlplane docker images](#build-controlplane-docker-images)
    - [Creating the Kubernetes cluster with kubeadm](#creating-the-kubernetes-cluster-with-kubeadm)
    - [Testing the Kubernetes cluster](#testing-the-kubernetes-cluster)
        - [Smoke test](#smoke-test)
            - [DNS](#dns)
            - [Deployments](#deployments)
            - [Port Forwarding](#port-forwarding)
            - [Logs](#logs)
            - [Exec](#exec)
            - [Services](#services)
        - [Conformance test](#conformance-test)
    - [Tips and Tricks](#tips-and-tricks)
        - [Semantic version ordering](#semantic-version-ordering)
        - [How to identify exact version/build number for a PR](#how-to-identify-exact-versionbuild-number-for-a-pr)
        - [Change the target version number when building a local release](#change-the-target-version-number-when-building-a-local-release)

<!-- /TOC -->

## Kubeadm version skew policy

When defining which pre-release versions to test, you should comply kubeadm version skew policy otherwise you will be blocked in the next phases.
See [Kubeadm version skew policy](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#version-skew-policy).

## Availability of pre-compiled release artifacts

The availability of pre-compiled deb/yum packages, kubeadm binary file or controlplane docker images depends by the Kubernetes build and deploy infrastructure.

The table below summarize the current state:

|                         | .deb or .rpm                                                 | kubeadm binary                                               | control plane                                                |
| ----------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| **GA release**          | from .deb or .rpm repository                                 | from [github release page](https://github.com/kubernetes/kubernetes/releases) or from `gs://kubernetes-release/release/` GCS bucket | from `k8s.gcr.io` container registry or from [github release page](https://github.com/kubernetes/kubernetes/releases)           |
| **alpha/beta release*** | not available. use CI/CD version "near" tag                  | from [github release page](https://github.com/kubernetes/kubernetes/releases) or from `gs://kubernetes-release/release/` GCS bucket | from `k8s.gcr.io` container registry or from [github release page](https://github.com/kubernetes/kubernetes/releases)        |
| **CI/CD release***      | from `gs://k8s-release-dev/bazel/` GCS bucket (only debs, built every merge) | from `gs://k8s-release-dev/ci-cross/` GCS bucket (built every merge) | from `gcr.io/kubernetes-ci-images` container registry (built every few hours, not by PR) |

[*] for alpha/beta and CI/CD currently it is not possible to have exact version number consistency for all the
components; however you can select version numbers "near to" the desired version.

To access GCS buckets from the command-line, install the [gsutil](https://cloud.google.com/storage/docs/gsutil_install) tool.

### Getting .deb or .rpm packages form repository

Pre-compiled GA version of .deb or .rpm packages are deployed into official Kubernetes repositories. See [installing kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl)
for instruction about how add the necessary repository and repository key.

To explore versions available in a repository use:

```bash
# Ubuntu, Debian or HypriotOS
apt-cache madison <package name>

# CentOS, RHEL or Fedora
yum --showduplicates list <package name>
```

To retrieve and install packages for a specific version use:

```bash
# Ubuntu, Debian or HypriotOS
apt-get install <package name>=<version number>

# CentOS, RHEL or Fedora
yum install <package name>-<version number>
```

### Getting .deb or .rpm packages form a GCS bucket

Pre-compiled CI/CD releases of .deb or .rpm packages are deployed into the `gs://k8s-release-dev/bazel/`
GCS bucket.

To explore versions available in Google Storage buckets use:

```bash
gsutil ls gs://k8s-release-dev/bazel/{filter}

# e.g. search all CI/CD v1.10 releases
gsutil ls -g gs://k8s-release-dev/bazel/v1.10*
```

As alternative, you can browse using <https://console.cloud.google.com/storage/browser/k8s-release-dev/bazel/> .

To retrieve a pre-compiled  CI/CD releases of .deb or .rpm version of kubeadm binary use:

```bash
gsutil cp gs://{bucket-name}/{release}/bin/linux/amd64 *.deb .
```

### Getting kubeadm binaries from a GCS bucket

Pre-compiled GA, alpha/beta versions of kubeadm binary are deployed into `gs://kubernetes-release/release/` GCS bucket,
while CI/CD versions are deployed into `gs://k8s-release-dev/ci-cross/` bucket.

To explore versions available in Google Storage buckets use:

```bash
gsutil ls gs://{bucket-name}{filter}

# e.g. search all GA and alpha/beta v1.10 releases
gsutil ls -d gs://kubernetes-release/release/v1.10*
```

As alternative, you can browse GCS buckets using <https://console.cloud.google.com/storage/browser/{bucket-name}> or
<http://gcsweb.k8s.io/gcs/kubernetes-release/release/> (only for `gs://kubernetes-release/release/`).

To retrieve a pre-compiled version of kubeadm binary use:

```bash
gsutil cp gs://{bucket-name}/{release}/bin/linux/arm64/kubeadm .

# or, only for releases in gs://kubernetes-release/release/
curl -L https://dl.k8s.io/release/{release}/bin/linux/amd64/kubeadm && chown +x kubeadm
```

### Getting docker images from a GCR registry

Kubeadm will take care of downloading images corresponding to the `--kubernetes-version` flag of kubeadm or the `kubernetesVersion` field of the [kubeadm config file](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init).

For GA, alpha/beta valid version numbers are:

- Semantic version number
- Kubernetes release labels for GA, alpha/beta versions like e.g. `stable`, `stable-1`, `stable-1.10`, `latest`,
  `latest-1`, `latest-1.10`. See <https://console.cloud.google.com/storage/browser/kubernetes-release/release/>
  for the full list of labels.

For CI/CD valid version numbers are:

- prefix `ci/` followed by a semantic version number
- prefix `ci/` followed by Kubernetes release labels for CI/CD versions like e.g. `ci/latest`, `ci/latest-1`, `latest-1.10`.
  See <https://console.cloud.google.com/storage/browser/k8s-release-dev/ci-cross/> for the full list of labels.

If you want to retrieve manually pre-compiled/pre-built GA, alpha/beta versions of control plane images, such
images are deployed into `k8s.gcr.io` GCR registry, while CI/CD versions are deployed into
`gcr.io/kubernetes-ci-images` GCR registry.

To explore versions available in Google Container Registry use

```bash
gcloud container images list-tags gcr.io/{gcr-repository-name}/{image-name}
```

Valid image names are `kube-apiserver-amd64`, `pause-amd64`, `etcd-amd64` etc. e.g.

```bash
gcloud container images list-tags k8s.gcr.io/kube-apiserver-amd64 --sort-by=~tags --filter=tags:v1.10.2 --limit=50
```

As alternative, you can browse <https://console.cloud.google.com/gcr/images/{gcr-repository-name}/GLOBAL/{image-name}>

To retrieve a pre-compiled version of control plane images `docker pull {image tag}`

### Getting kubeadm binaries or docker images form github release page

Pre-compiled version of Kubeadm binaries and docker images for GA and alpha/beta versions can be retrieved form the
[github release page](https://github.com/kubernetes/kubernetes/releases).

Server binary and tarball are no longer included in the `kubernetes.tar.gz` package.
Run `cluster/get-kube-binaries.sh` to download the tarball with server binaries.

> Inside release notes, usually there is a direct link for getting server binaries directly

> `cluster/get-kube-binaries.sh` retrieves server binaries from `gs://kubernetes-release/release/{release}`
  GCS bucket; you can use `gsutil` to get server binaries directly.

Both Kubeadm binaries and docker images are available in `/server/bin` folder of  `kubernetes-server-linux-amd64.tar.gz`

## Create a local version

A local version of all the release artifacts (.debs or .rpm, kubeadm binary, docker images) can be build locally; build
instructions are provided for bazel only, but other builds methods supported by Kuberentes can be used as well.

See also:

- [Build and test with Bazel](https://git.k8s.io/community/contributors/devel/sig-testing/bazel.md)
- [Change the target version number when building a local release](#change-the-target-version-number-when-building-a-local-release)

### Build .debs packages

```bash
cd ~/go/src/k8s.io/kubernetes

# build debs
bazel build //build/debs
```

build output will be stored in `bazel-bin/build/debs`.

> cross build not supported yet; the unofficial [Planter tool](https://git.k8s.io/test-infra/planter)
> can be used to overcome the problem

> currently bazel does not provide target for building rpm packages

### Build kubeadm binary

```bash
cd ~/go/src/k8s.io/kubernetes

# run all unit tests
bazel test //cmd/kubeadm/...

# build kubeadm binary for the target platform used by the machines in the playground (linux)
bazel build //cmd/kubeadm

# To cross build for linux- amd64 from mac
bazel build //cmd/kubeadm --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
```

build output will be stored in  `bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm`.

### Build controlplane docker images

```bash
cd ~/go/src/k8s.io/kubernetes

# build docker images
bazel build //build:docker-artifacts
```

> cross build not supported yet; the unofficial Planter tool can be used to overcome the problem

build output will be stored in  `bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm`.

## Creating the Kubernetes cluster with kubeadm

The kubernetes web site contains reference documentation for [installing kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/)
and for [creating a cluster using it](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/).

According to selected versions to be installed (the version of .deb or .rpm packages, the version of the kubeadm binary to be used, the version number of the control plane components) following variants to the standard procedure should be applied:

- if you are going to use .deb or .rpm packages available locally (no matter if retrieved from GCS buckets pre-compiled or built locally), you should not
  execute action described in [Installing kubeadm, kubelet and kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl). Instead you should e.g.

  ```bash
  # Ubuntu, Debian or HypriotOS (assuming the command executed in the same folder of deb files)
  sudo apt install path/to/kubectl.deb path/to/kubeadm.deb path/to/kubelet.deb path/to/kubernetes-cni.deb
  ````

- if you are going to use a kubeadm binary available locally, overriding the one installed with packages, after
  [Installing kubeadm, kubelet and kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl) you should

  ```bash
  cp path/to/local/kubeadm /usr/bin/kubeadm
  ```

  or simply use `path/to/local/kubeadm` instead of `kubeadm` in following steps.

- if you are going to use control plane images available locally, before executing [kubeadm init](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#instructions) you should

  ```bash
  array=(kube-apiserver kube-controller-manager kube-scheduler kube-proxy)
  for i in "${array[@]}"; do
    sudo docker load -i path/to/$i.tar
  done
  ```

  Then, you should take care of passing the exact version of your images to `kubeadm init` using `--kubernetes-version` flag
  or the `kubernetesVersion` field of the [kubeadm config file](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init).

## Testing the Kubernetes cluster

### Smoke test

(All credits to Kelsey Hightower - [Kubernetes The Hard Way](https://github.com/kelseyhightower/kubernetes-the-hard-way))

#### DNS

In this section you will verify the proper functioning of [DNS for Services and Pods](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/).

Create a `busybox` deployment:

```bash
kubectl run busybox --image=busybox --command -- sleep 3600
```

List the pod created by the `busybox` deployment:

```bash
kubectl get pods -l run=busybox
```

> output

```bash
NAME                       READY     STATUS    RESTARTS   AGE
busybox-2125412808-mt2vb   1/1       Running   0          15s
```

Retrieve the full name of the `busybox` pod:

```bash
POD_NAME=$(kubectl get pods -l run=busybox -o jsonpath="{.items[0].metadata.name}")
```

Execute a DNS lookup for the `kubernetes` service inside the `busybox` pod:

```bash
kubectl exec -ti $POD_NAME -- nslookup kubernetes
```

> output

```bash
Server:    10.32.0.10
Address 1: 10.32.0.10 kube-dns.kube-system.svc.cluster.local

Name:      kubernetes
Address 1: 10.32.0.1 kubernetes.default.svc.cluster.local
```

#### Deployments

In this section you will verify the ability to create and manage [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/).

Create a deployment for the [nginx](https://nginx.org/en/) web server:

```bash
kubectl run nginx --image=nginx
```

List the pod created by the `nginx` deployment:

```bash
kubectl get pods -l run=nginx
```

> output

```bash
NAME                     READY     STATUS    RESTARTS   AGE
nginx-65899c769f-xkfcn   1/1       Running   0          15s
```

#### Port Forwarding

In this section you will verify the ability to access applications remotely using [port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/).

Retrieve the full name of the `nginx` pod:

```bash
POD_NAME=$(kubectl get pods -l run=nginx -o jsonpath="{.items[0].metadata.name}")
```

Forward port `8080` on your local machine to port `80` of the `nginx` pod:

```bash
kubectl port-forward $POD_NAME 8080:80
```

> output

```bash
Forwarding from 127.0.0.1:8080 -> 80
Forwarding from [::1]:8080 -> 80
```

In a new terminal make an HTTP request using the forwarding address:

```bash
curl --head http://127.0.0.1:8080
```

> output

```bash
HTTP/1.1 200 OK
Server: nginx/1.13.12
Date: Mon, 14 May 2018 13:59:21 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Mon, 09 Apr 2018 16:01:09 GMT
Connection: keep-alive
ETag: "5acb8e45-264"
Accept-Ranges: bytes
```

Switch back to the previous terminal and stop the port forwarding to the `nginx` pod:

```bash
Forwarding from 127.0.0.1:8080 -> 80
Forwarding from [::1]:8080 -> 80
Handling connection for 8080
^C
```

#### Logs

In this section you will verify the ability to [retrieve container logs](https://kubernetes.io/docs/concepts/cluster-administration/logging/).

Print the `nginx` pod logs:

```bash
kubectl logs $POD_NAME
```

> output

```bash
127.0.0.1 - - [14/May/2018:13:59:21 +0000] "HEAD / HTTP/1.1" 200 0 "-" "curl/7.52.1" "-"
```

#### Exec

In this section you will verify the ability to [execute commands in a container](https://kubernetes.io/docs/tasks/debug-application-cluster/get-shell-running-container/#running-individual-commands-in-a-container).

Print the nginx version by executing the `nginx -v` command in the `nginx` container:

```bash
kubectl exec -ti $POD_NAME -- nginx -v
```

> output

```bash
nginx version: nginx/1.13.12
```

#### Services

In this section you will verify the ability to expose applications using a [Service](https://kubernetes.io/docs/concepts/services-networking/service/).

Expose the `nginx` deployment using a [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport) service:

```bash
kubectl expose deployment nginx --port 80 --type NodePort
```

> The LoadBalancer service type can not be used because your cluster is not configured with [cloud provider integration](https://kubernetes.io/docs/getting-started-guides/scratch/#cloud-provider). Setting up cloud provider integration is out of scope for this tutorial.

Retrieve the node port assigned to the `nginx` service:

```bash
NODE_PORT=$(kubectl get svc nginx \
  --output=jsonpath='{range .spec.ports[0]}{.nodePort}')
```

Create a firewall rule that allows remote access to the `nginx` node port:

Retrieve the external IP address of a worker instance and make an HTTP request using the external
IP address and the `nginx` node port:

```bash
curl -I http://${EXTERNAL_IP}:${NODE_PORT}
```

> output

```bash
HTTP/1.1 200 OK
Server: nginx/1.13.12
Date: Mon, 14 May 2018 14:01:30 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Mon, 09 Apr 2018 16:01:09 GMT
Connection: keep-alive
ETag: "5acb8e45-264"
Accept-Ranges: bytes
```

### Conformance test

(all credits to [Heptio](https://heptio.com/))

By using [Heptio Sonobuoy](https://heptio.com/products/#heptio-sonobuoy) it is possible to ensure that a cluster is
properly configured and that its behavior conforms to official Kubernetes specifications.

[Sonobuoy Scanner](https://scanner.heptio.com/) provides a browser-based interface to help you install and run Sonobuoy;
it gives you the Sonobuoy command to run, and then automatically reports whether the tests succeed.

## Tips and Tricks

### Semantic version ordering

see [https://semver.org/](https://semver.org/) for full explanation. Briefly:

- MAJOR.MINOR.PATCH are non-negative integers and use numerical ordering
- Pre-release versions have a lower precedence than the associated normal version; pre-release versions use
  alphanumerical ordering
- Build metadata should be ignored when determining version precedence

### How to identify exact version/build number for a PR

1. Open one PR already merged e.g. one well known PR, the last PR merged on master, the last PR merged yesterday or
  the last PR merged on a specific branch/tag
2. Scroll down and identify the *"k8s-merge-robot merged"* comment; click `View Details`, and then `details`
  for the`pull-kubernetes-bazel-build` Job.
3. In the resulting web page, you can then grab the {pr-build-id} which is a string like e.g.
 `master:dc02dfe5604e17f3b0f1f7cafb03597298ef0e3f,52201:79725246f401d0b8524f4c96bdc09535f3037c68`
4. Then open in the browser the corresponding bucket using the address
  <https://console.cloud.google.com/storage/browser/kubernetes-jenkins/shared-results/{pr-build-id};>
5. This bucket should contains a file named `bazel-build-location.txt`; inside there is the name of the GCS bucket
  where CI/CD output are stored by jenkins e.g. `gs://k8s-release-dev/bazel/v1.9.0-alpha.0.584+0bfca758a870fc`

### Change the target version number when building a local release

It is also possible to force the target version before building by creating a file that sets environment
variables specifying the desired version, e.g. `$working_dir/buildversion` with:

```bash
export KUBE_GIT_MAJOR=1
export KUBE_GIT_MINOR=11
export KUBE_GIT_VERSION=v1.11.1
export KUBE_GIT_TREE_STATE=clean
export KUBE_GIT_COMMIT=d34db33f
```

And then this file should be passed to the build process via another env variable

```bash
export KUBE_GIT_VERSION_FILE=$working_dir/buildversion
```

> use `hack/print-workspace-status.sh` to check the target version before build.
