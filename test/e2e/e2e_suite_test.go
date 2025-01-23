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
