package sdnclient

import (
	"io"
	"net/http"
	"strings"
	"testing"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
)

type networkCheckTransport func(*http.Request) (*http.Response, error)

func (f networkCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNetworkExistenceDoesNotTreatServerErrorsAsAbsence(t *testing.T) {
	for _, status := range []int{200, 404, 401, 503} {
		session := NewSessionClient("http://onos.invalid/onos", "karaf", "karaf")
		session.httpClient.Transport = networkCheckTransport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		})
		c := &InternalClient{Session: session}
		exists, err := c.CheckNetworkExists(l2smv1.NetworkTypeVnet, "network")
		if status == 200 && (!exists || err != nil) {
			t.Fatal("existing network not recognized")
		}
		if status == 404 && (exists || err != nil) {
			t.Fatal("missing network not recognized")
		}
		if status != 200 && status != 404 && err == nil {
			t.Fatalf("HTTP %d silently treated as a missing network", status)
		}
	}
}
