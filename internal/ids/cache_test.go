package ids

import (
	"testing"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestSuricataUsesCachedImage(t *testing.T) {
	deployment := generateSuricataDeployment(&l2smv1.IdsRules{Node: "ids-node"}, "test-network", "[]", "test")
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("cached IDS image would require registry access: %s", container.ImagePullPolicy)
	}
}
