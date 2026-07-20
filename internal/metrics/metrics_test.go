package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ishioni/dexd/internal/config"
)

func TestHandlerExportsMetrics(t *testing.T) {
	SetBuildInfo("test-version", "test-sha", config.PolicySync)
	SetPlanMetrics(map[string]int{"A": 1, "CNAME": 1}, map[string]int{"create": 1})
	IncDockerEvent("start")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "dexd_build_info") {
		t.Fatalf("metrics body missing build_info: %s", body)
	}
	if !strings.Contains(body, "dexd_docker_events_total") {
		t.Fatalf("metrics body missing docker_events_total: %s", body)
	}
	if !strings.Contains(body, "dexd_plan_desired_records") {
		t.Fatalf("metrics body missing plan_desired_records: %s", body)
	}
	if !strings.Contains(body, "dexd_plan_desired_records{record_type=\"A\"}") ||
		!strings.Contains(body, "dexd_plan_desired_records{record_type=\"CNAME\"}") ||
		!strings.Contains(body, "dexd_plan_desired_records{record_type=\"TXT\"}") {
		t.Fatalf("metrics body missing plan_desired_records by type: %s", body)
	}
	if !strings.Contains(body, "dexd_plan_desired_records{record_type=\"TXT\"} 2") {
		t.Fatalf("metrics body has an incorrect TXT ownership record count: %s", body)
	}
	if strings.Contains(body, "dexd_plan_current_records") {
		t.Fatalf("metrics body unexpectedly exposes plan_current_records: %s", body)
	}
}
