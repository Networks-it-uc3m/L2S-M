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

package controller

import (
	"context"
	"fmt"

	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
)

var _ = Describe("QuarantinePodRequest Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}

		BeforeEach(func() {
			createL2Network(ctx, "source-network", map[string]string{"l2sm/network": "production"}, 1)
			createL2Network(ctx, "quarantine-network", nil, 0)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ping",
					Namespace: "default",
					Labels: map[string]string{
						"app": "ping",
					},
					Annotations: map[string]string{
						networkannotation.L2SM_NETWORK_ANNOTATION: `[{"name":"source-network"}]`,
						networkannotation.MULTUS_ANNOTATION_KEY:   `[{"name":"l2sm-veth1","ips":["10.0.0.2/24"]}]`,
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "node-a",
					Containers: []corev1.Container{
						{Name: "ping", Image: "busybox"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			resource := &l2smv1.QuarantinePodRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: l2smv1.QuarantinePodRequestSpec{
					Selector: l2smv1.QuarantinePodSelector{
						PodLabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "ping"},
						},
						L2NetworkSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"l2sm/network": "production"},
						},
					},
					TargetL2Network: "quarantine-network",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			deleteIfExists(ctx, &l2smv1.QuarantinePodRequest{}, typeNamespacedName)
			deleteIfExists(ctx, &corev1.Pod{}, types.NamespacedName{Name: "ping", Namespace: "default"})
			deleteIfExists(ctx, &l2smv1.L2Network{}, types.NamespacedName{Name: "source-network", Namespace: "default"})
			deleteIfExists(ctx, &l2smv1.L2Network{}, types.NamespacedName{Name: "quarantine-network", Namespace: "default"})
		})
		It("moves the selected pod from the source network to the target network", func() {
			By("Reconciling the created resource")
			fakeSDN := &fakeQuarantineSDNClient{existingNetworks: map[string]bool{"quarantine-network": true}}
			controllerReconciler := &QuarantinePodRequestReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				InternalClient: fakeSDN,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSDN.calls).To(HaveLen(3))
			Expect(fakeSDN.calls[0]).To(Equal("check:quarantine-network"))
			Expect(fakeSDN.calls[1]).To(HavePrefix("detach:source-network:"))
			Expect(fakeSDN.calls[2]).To(HavePrefix("attach:quarantine-network:"))

			pod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "ping", Namespace: "default"}, pod)).To(Succeed())
			Expect(pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION]).To(ContainSubstring("quarantine-network"))
			Expect(pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION]).NotTo(ContainSubstring("source-network"))

			request := &l2smv1.QuarantinePodRequest{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, request)).To(Succeed())
			available := meta.FindStatusCondition(request.Status.Conditions, "Available")
			Expect(available).NotTo(BeNil())
			Expect(available.Status).To(Equal(metav1.ConditionTrue))
			Expect(request.Status.SourceL2NetworkName).To(Equal("source-network"))
			Expect(request.Status.TargetL2NetworkName).To(Equal("quarantine-network"))
			Expect(request.Status.MovedPodCount).To(Equal(int32(1)))
		})
	})
})

type fakeQuarantineSDNClient struct {
	existingNetworks map[string]bool
	calls            []string
}

func (c *fakeQuarantineSDNClient) CreateNetwork(l2smv1.NetworkType, interface{}) error {
	return nil
}

func (c *fakeQuarantineSDNClient) DeleteNetwork(l2smv1.NetworkType, string) error {
	return nil
}

func (c *fakeQuarantineSDNClient) CheckNetworkExists(_ l2smv1.NetworkType, networkID string) (bool, error) {
	c.calls = append(c.calls, fmt.Sprintf("check:%s", networkID))
	return c.existingNetworks[networkID], nil
}

func (c *fakeQuarantineSDNClient) AttachPodToNetwork(_ l2smv1.NetworkType, config interface{}) error {
	payload := config.(sdnclient.VnetPayload)
	c.calls = append(c.calls, fmt.Sprintf("attach:%s:%s", payload.NetworkId, payload.Port[0]))
	return nil
}

func (c *fakeQuarantineSDNClient) DetachPodFromNetwork(_ l2smv1.NetworkType, config interface{}) error {
	payload := config.(sdnclient.VnetPayload)
	c.calls = append(c.calls, fmt.Sprintf("detach:%s:%s", payload.NetworkId, payload.Port[0]))
	return nil
}

func (c *fakeQuarantineSDNClient) SetUpMirrorPort(l2smv1.NetworkType, any) error {
	return nil
}

func createL2Network(ctx context.Context, name string, labels map[string]string, connectedPodCount int) {
	network := &l2smv1.L2Network{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: l2smv1.L2NetworkSpec{
			Type: l2smv1.NetworkTypeVnet,
		},
	}
	Expect(k8sClient.Create(ctx, network)).To(Succeed())
	network.Status.ConnectedPodCount = connectedPodCount
	network.Status.AssignedIPs = map[string]string{"10.0.0.2": "ping"}
	Expect(k8sClient.Status().Update(ctx, network)).To(Succeed())
}

func deleteIfExists(ctx context.Context, obj client.Object, key types.NamespacedName) {
	err := k8sClient.Get(ctx, key, obj)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
}
