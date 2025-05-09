// Copyright 2024 Universidad Carlos III de Madrid
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// client-go or controller-runtime
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient  crclient.Client
	testScheme = runtime.NewScheme()
	restConfig *rest.Config
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Test Suite")
}

// BeforeSuite: set up real kube client, NAD CRD scheme, etc.
var _ = BeforeSuite(func() {
	// Let’s parse flags to avoid noisy logs from klog, glog, etc.
	flag.Parse()

	// Resolve the kubeconfig path
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	// Build the *rest.Config from kubeconfig
	var err error
	restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).NotTo(HaveOccurred(), "failed to build rest.Config from kubeconfig")

	// Add NAD types to Scheme so controller-runtime knows how to decode them
	err = nettypes.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred(), "failed to add NAD CRDs to scheme")

	// Create a controller-runtime client with the scheme
	k8sClient, err = crclient.New(restConfig, crclient.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create controller-runtime client")
})
