package webui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPanelDirectQuotaEditReachesDirectSettings(t *testing.T) {
	for _, limit := range []int64{0, 5} {
		body, err := json.Marshal(map[string]any{
			"name": "example", "direct": map[string]any{"trafficLimitGB": limit},
		})
		if err != nil {
			t.Fatal(err)
		}
		var req tunnelEditRequest
		dec := json.NewDecoder(strings.NewReader(string(body)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Direct.TrafficLimitGB == nil || *req.Direct.TrafficLimitGB != limit {
			t.Fatalf("quota %d decoded as %v", limit, req.Direct.TrafficLimitGB)
		}
	}
}
