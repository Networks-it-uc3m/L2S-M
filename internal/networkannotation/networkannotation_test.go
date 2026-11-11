package networkannotation

import "testing"

func TestExtractNetworksPreservesExplicitNamespace(t *testing.T) {
	for _, fallback := range []string{"", "workloads"} {
		networks, err := ExtractNetworks(`[{"name":"overlay-veth1","namespace":"switches"}]`, fallback)
		if err != nil || len(networks) != 1 || networks[0].Namespace != "switches" {
			t.Fatalf("explicit namespace overwritten by %q: %+v %v", fallback, networks, err)
		}
	}
}
func TestExtractNetworksDefaultsOnlyMissingNamespace(t *testing.T) {
	for _, raw := range []string{`[{"name":"overlay-veth1"}]`, `overlay-veth1`} {
		networks, err := ExtractNetworks(raw, "default")
		if err != nil || len(networks) != 1 || networks[0].Namespace != "default" {
			t.Fatalf("missing namespace not defaulted: %+v %v", networks, err)
		}
	}
}
