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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// Import the function under test.  Adjust import path to your repo layout!
	"github.com/Networks-it-uc3m/L2S-M/internal/controller"
)

var _ = Describe("Pod Utils E2E", func() {

	var (
		ctx               context.Context
		labelKey          string
		switchesNamespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		labelKey = "used-overlay-test"
		switchesNamespace = "test-namespace"

		By("creating the test namespace (or ignoring if it exists)")
		err := createNamespace(ctx, switchesNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("creating the other namespace (or ignoring if it exists)")
		err = createNamespace(ctx, "other-namespace")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return free NADs matching the conditions", func() {
		// 	// 1. Define test objects
		// 	nad1 := &nettypes.NetworkAttachmentDefinition{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "nad1-e2e",
		// 			Namespace: switchesNamespace,
		// 			Labels: map[string]string{
		// 				"app":    "l2sm",
		// 				labelKey: "true",
		// 			},
		// 		},
		// 	}
		// 	nad2 := &nettypes.NetworkAttachmentDefinition{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "nad2-e2e",
		// 			Namespace: switchesNamespace,
		// 			Labels: map[string]string{
		// 				"app":    "l2sm",
		// 				labelKey: "false",
		// 			},
		// 		},
		// 	}
		// 	nad3 := &nettypes.NetworkAttachmentDefinition{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "nad3-e2e",
		// 			Namespace: switchesNamespace,
		// 			Labels: map[string]string{
		// 				"app": "l2sm",
		// 			},
		// 		},
		// 	}
		// 	nad4 := &nettypes.NetworkAttachmentDefinition{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "nad4-e2e",
		// 			Namespace: "other-namespace",
		// 			Labels: map[string]string{
		// 				"app":    "l2sm",
		// 				labelKey: "true",
		// 			},
		// 		},
		// 	}

		// 2. Create them in the cluster
		// By("creating NAD objects in the cluster")
		// Expect(k8sClient.Create(ctx, nad1)).To(Succeed())
		// Expect(k8sClient.Create(ctx, nad2)).To(Succeed())
		// Expect(k8sClient.Create(ctx, nad3)).To(Succeed())
		// Expect(k8sClient.Create(ctx, nad4)).To(Succeed())

		// 3. Invoke the function under test
		//    We call GetFreeNetAttachDefs, but passing our real k8sClient, real namespace, etc.
		result := controller.GetFreeNetAttachDefs(ctx, k8sClient, switchesNamespace, labelKey)

		// 4. Debug print
		for _, item := range result.Items {
			fmt.Printf("NAD: %s Labels: %+v\n", item.Name, item.Labels)
		}

		// 5. Verify expectations
		Expect(len(result.Items)).To(BeNumerically("==", 2), "Expected exactly 2 matching NADs")
		names := []string{result.Items[0].Name, result.Items[1].Name}
		Expect(names).To(ContainElements("nad2-e2e", "nad3-e2e"))
	})
})

// createNamespace is a helper that tries to create a namespace, ignoring if it already exists.
func createNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := k8sClient.Create(ctx, ns)
	if err != nil {
		// Maybe the namespace already exists; you can handle that gracefully if you like
	}
	return nil
}
