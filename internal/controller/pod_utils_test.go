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

// generate_test.go
package controller

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGenerate4byteChunk verifies that Generate4byteChunk produces a 4-character hexadecimal string.
// func TestGenerate4byteChunk(t *testing.T) {
// 	// Regular expression to match exactly 4 hexadecimal characters
// 	re := regexp.MustCompile(`^[0-9a-fA-F]{4}$`)

// 	// Call Generate4byteChunk multiple times to check if output is always 4 characters
// 	for i := 0; i < 100; i++ {
// 		output := Generate4byteChunk()

// 		// Check if the output matches the 4-character hex pattern
// 		if !re.MatchString(output) {
// 			t.Errorf("Expected a 4-character hexadecimal string, but got: %s", output)
// 		}
// 	}
// }

var _ = Describe("Pod Utils", func() {

	// -------------------------------------------------------------------------
	// TestGenerate4byteChunk as a Ginkgo It()
	// -------------------------------------------------------------------------
	Describe("Generate4byteChunk", func() {
		It("should produce a 4-character hexadecimal string", func() {
			re := regexp.MustCompile(`^[0-9a-fA-F]{4}$`)
			for i := 0; i < 100; i++ {
				output := Generate4byteChunk()
				Expect(re.MatchString(output)).To(BeTrue(),
					"Expected a 4-character hex string, but got: %s", output)
			}
		})
	})

	// -------------------------------------------------------------------------
	// TestGetFreeNetAttachDefs as a Ginkgo It()
	// -------------------------------------------------------------------------
	Describe("GetFreeNetAttachDefs", func() {
		var (
			ctx               context.Context
			labelKey          string
			switchesNamespace string
		)

		BeforeEach(func() {
			ctx = context.Background()
			labelKey = "used-overlay-test"
			switchesNamespace = "test-namespace"

			// Create or ensure the namespace exists
			By("creating the test namespace")
			err := createNamespace(ctx, switchesNamespace)
			Expect(err).NotTo(HaveOccurred())

			// Create or ensure the namespace exists
			By("creating the other namespace")
			err = createNamespace(ctx, "other-namespace")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return free NADs matching the conditions", func() {
			// 1. Define test objects
			nad1 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad1",
					Namespace: switchesNamespace,
					Labels: map[string]string{
						"app":    "l2sm",
						labelKey: "true",
					},
				},
			}
			nad2 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad2",
					Namespace: switchesNamespace,
					Labels: map[string]string{
						"app":    "l2sm",
						labelKey: "false",
					},
				},
			}
			nad3 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad3",
					Namespace: switchesNamespace,
					Labels: map[string]string{
						"app": "l2sm",
					},
				},
			}
			nad4 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad4",
					Namespace: "other-namespace",
					Labels: map[string]string{
						"app":    "l2sm",
						labelKey: "true",
					},
				},
			}

			// 2. Create them in the cluster
			By("creating NAD objects in the cluster")
			Expect(k8sClient.Create(ctx, nad1)).To(Succeed())
			Expect(k8sClient.Create(ctx, nad2)).To(Succeed())
			Expect(k8sClient.Create(ctx, nad3)).To(Succeed())
			Expect(k8sClient.Create(ctx, nad4)).To(Succeed())

			// 3. Invoke the function under test
			result := GetFreeNetAttachDefs(ctx, k8sClient, switchesNamespace, labelKey)

			// 4. (Optional) debug
			for _, item := range result.Items {
				fmt.Printf("NAD: %s Labels: %+v\n", item.Name, item.Labels)
			}

			// 5. Verify expectations
			Expect(len(result.Items)).To(BeNumerically("==", 2), "Expected exactly 2 matching NADs")
			// Typically it's nad2 and nad3
			names := []string{result.Items[0].Name, result.Items[1].Name}
			Expect(names).To(ContainElements("nad2", "nad3"))
		})
	})
})

func createNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return k8sClient.Create(ctx, ns)
}
