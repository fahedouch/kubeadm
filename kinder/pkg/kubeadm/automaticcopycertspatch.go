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
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/constants"
)

// GetAutomaticCopyCertsPatches returns the kubeadm config patch that will instruct kubeadm
// to use a well known certificate key for init/join.
func GetAutomaticCopyCertsPatches(kubeadmConfigVersion string) ([]string, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing automaticCopyCertsPatches for kubeadm config %s", kubeadmConfigVersion)

	switch kubeadmConfigVersion {
	case "v1beta2":
		return []string{
			fmt.Sprintf(automaticCopyCertsInitv1beta2, constants.CertificateKey),
			fmt.Sprintf(automaticCopyCertsJoinv1beta2, constants.CertificateKey),
		}, nil
	case "v1beta3":
		return []string{
			fmt.Sprintf(automaticCopyCertsInitv1beta3, constants.CertificateKey),
			fmt.Sprintf(automaticCopyCertsJoinv1beta3, constants.CertificateKey),
		}, nil
	}

	return nil, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
}

const automaticCopyCertsInitv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
metadata:
  name: config
certificateKey: "%s"`

const automaticCopyCertsJoinv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
controlPlane:
  certificateKey: "%s"`

const automaticCopyCertsInitv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
metadata:
  name: config
certificateKey: "%s"`

const automaticCopyCertsJoinv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
metadata:
  name: config
controlPlane:
  certificateKey: "%s"`
