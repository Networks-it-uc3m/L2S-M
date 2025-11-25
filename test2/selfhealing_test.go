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

// selfhealing_test.go
package selfhealing_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

/* ---------- Configurable knobs ---------- */
const (
	controllerBaseURL = "http://172.18.0.4:30000/onos"
	networkTypePath   = "vnets"
	selfHeals         = 10              // how many reconfigurations we try
	requestTimeout    = 5 * time.Second // per request
	userPass          = "karaf:karaf"   // basic-auth credentials
)

/* ---------- Wire model ---------- */
type payload struct {
	NetworkID string   `json:"networkId"`
	Port      []string `json:"networkEndpoints,omitempty"`
}

/* ---------- Test ---------- */
func TestSelfHealingNetwork(t *testing.T) {
	t.Log("📋  Self-Healing Network test starting")

	// ---------- 1. Static construction ----------
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(userPass))
	url := fmt.Sprintf("%s/%s/api/port", controllerBaseURL, networkTypePath)
	t.Logf("⏩  Controller endpoint: %s", url)

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Timeout: requestTimeout, Transport: transport}

	// Pre-build the 10 distinct payloads and JSON blobs
	payloadBytes := make([][]byte, selfHeals)
	for i := 0; i < selfHeals; i++ {
		pl := payload{
			NetworkID: fmt.Sprintf("test-network-%d", i+1),
			Port:      []string{fmt.Sprintf("of:63b9c50cd5f12d25/%d", i+1)},
		}
		b, err := json.Marshal(pl)
		if err != nil {
			t.Fatalf("marshal payload %d: %v", i, err)
		}
		payloadBytes[i] = b
	}
	t.Logf("✅  Prepared %d unique payloads", selfHeals)

	// ---------- 2. Fire concurrent requests ----------
	var wg sync.WaitGroup
	wg.Add(selfHeals)

	durations := make([]time.Duration, selfHeals)
	startWall := time.Now()

	for i := 0; i < selfHeals; i++ {
		go func(idx int) {
			defer wg.Done()
			iterStart := time.Now()

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes[idx]))
			if err != nil {
				t.Errorf("build req %d: %v", idx, err)
				return
			}
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("do req %d: %v", idx, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				t.Errorf("req %d: unexpected HTTP %d", idx, resp.StatusCode)
				return
			}

			durations[idx] = time.Since(iterStart)
			t.Logf("🔧  Self-heal %2d finished in %s", idx+1, durations[idx])
		}(i)
	}

	wg.Wait()
	wall := time.Since(startWall)

	// ---------- 3. Summarise ----------
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := sum / time.Duration(selfHeals)

	t.Log("📊  -------- summary --------")
	t.Logf("Total wall-clock time : %s", wall)
	t.Logf("Average per self-heal : %.2f ms", float64(avg.Microseconds())/1000)
	t.Log("✅  Self-Healing Network test completed")
}
