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

package status

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cri/host"
	"k8s.io/kubeadm/kinder/pkg/exec"
)

// NB. code implemented in this package ideally should be in the CRI package, but ATM it is
// implemented here to avoid circular references. TODO: refactor

// ContainerRuntime defines CRI runtime that are supported inside a kind(er) node
type ContainerRuntime string

const (
	// DockerRuntime refers to the docker container runtime
	DockerRuntime ContainerRuntime = "docker"
	// ContainerdRuntime refers to the containerd container runtime
	ContainerdRuntime ContainerRuntime = "containerd"
)

// InspectCRIinImage inspect an image and detects the installed container runtime
func InspectCRIinImage(image string) (ContainerRuntime, error) {
	// define docker default args
	id := "kind-detect-" + uuid.New().String()
	runArgs := []string{
		"-d", // make the client exit while the container continues to run
		"--entrypoint=sleep",
		"--name=" + id,
	}
	contatinerArgs := []string{"infinity"} // sleep infinitely to keep the container around

	if err := host.Run(image, runArgs, contatinerArgs); err != nil {
		return "", errors.Wrap(err, "error creating a temporary container for CRI detection")
	}
	defer func() {
		exec.NewHostCmd("docker", "rm", "-f", id).Run()
	}()

	return InspectCRIinContainer(id)
}

// InspectCRIinContainer inspect a running container and detects the installed container runtime
// NB. this method use raw kinddocker/kindexec commands because it is used also during "alter" and "create"
// (before an actual Cluster status exist)
func InspectCRIinContainer(id string) (ContainerRuntime, error) {
	lines, err := exec.NewNodeCmd(id, "/bin/sh", "-c", `which docker || true`).Silent().RunAndCapture()

	if err != nil {
		return ContainerRuntime(""), errors.Wrap(err, "error detecting CRI")
	}

	if len(lines) > 0 {
		return DockerRuntime, nil
	}

	return ContainerdRuntime, nil
}
