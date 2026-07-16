package gateway_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/uiansol/vigil-auditor/internal/gateway"
)

func TestHealthzWithoutDBReturns503(t *testing.T) {
	t.Parallel()

	app := gateway.NewAppForTest()
	req := httptest.NewRequest("GET", "/healthz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 503 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 503, got %d body=%s", resp.StatusCode, body)
	}
}
