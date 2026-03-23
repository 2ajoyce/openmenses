package mobile

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStartStop verifies basic engine startup and shutdown.
func TestStartStop(t *testing.T) {
	// Use a temporary database file.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := Start(dbPath, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	port := Port()
	token := AuthToken()

	if port <= 0 {
		t.Fatalf("invalid port: %d", port)
	}
	if len(token) != 64 {
		t.Fatalf("token should be 64 hex characters, got %d", len(token))
	}

	// Verify we can make a request to the server.
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/", port), nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()
	// We expect a 404 since no UI assets are mounted, but the request should go through.

	// Stop the engine.
	if err := Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// After Stop, Port/AuthToken return zero values.
	if Port() != 0 {
		t.Errorf("Port() should return 0 after Stop")
	}
	if AuthToken() != "" {
		t.Errorf("AuthToken() should return empty string after Stop")
	}

	// Second Stop should be idempotent.
	if err := Stop(); err != nil {
		t.Fatalf("second Stop failed: %v", err)
	}
}

// TestDoubleStart verifies that calling Start twice returns an error.
func TestDoubleStart(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := Start(dbPath, ""); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	defer Stop() //nolint:errcheck

	port1 := Port()
	token1 := AuthToken()

	// Second Start should fail.
	err := Start(filepath.Join(tmpDir, "other.db"), "")
	if err == nil {
		t.Fatalf("second Start should have failed")
	}
	if port1 <= 0 || token1 == "" {
		t.Fatalf("first Start should have succeeded: port=%d, token=%s", port1, token1)
	}
}

// TestAuthMiddleware verifies that auth tokens are properly validated on the
// Connect-RPC service path, and that static file routes are unauthenticated.
func TestAuthMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := Start(dbPath, ""); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop() //nolint:errcheck

	port := Port()
	token := AuthToken()

	httpClient := &http.Client{Timeout: 2 * time.Second}
	// Connect-RPC procedures are posted to /openmenses.v1.CycleTrackerService/<Method>.
	servicePath := fmt.Sprintf("http://127.0.0.1:%d/openmenses.v1.CycleTrackerService/GetUserProfile", port)

	// 1. No Authorization header → 401.
	req, _ := http.NewRequest("POST", servicePath, bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/connect+proto")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no auth header: expected 401, got %d", resp.StatusCode)
	}

	// 2. Wrong token → 401.
	req, _ = http.NewRequest("POST", servicePath, bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/connect+proto")
	req.Header.Set("Authorization", "Bearer "+"0000000000000000000000000000000000000000000000000000000000000000")
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token: expected 401, got %d", resp.StatusCode)
	}

	// 3. Correct token → non-401 (engine processes the request; empty body yields a Connect error, not 401).
	req, _ = http.NewRequest("POST", servicePath, bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/connect+proto")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("valid token: expected non-401, got %d", resp.StatusCode)
	}

	// 4. Static file route (/) does NOT require auth.
	req, _ = http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/", port), nil)
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("static route: should not require auth, got 401")
	}
}

// TestStaticFileServing verifies that static files are served and SPA routing works.
func TestStaticFileServing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	uiDir := filepath.Join(tmpDir, "ui")

	// Create UI directory with index.html.
	if err := os.MkdirAll(uiDir, 0755); err != nil {
		t.Fatalf("creating ui dir: %v", err)
	}
	indexPath := filepath.Join(uiDir, "index.html")
	indexContent := `<html><body>test content</body></html>`
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("writing index.html: %v", err)
	}

	if err := Start(dbPath, uiDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop() //nolint:errcheck

	port := Port()
	token := AuthToken()
	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Helper to make authenticated requests.
	makeReq := func(path string) (int, string) {
		req, _ := http.NewRequest("GET", baseURL+path, nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}

	// Test 1: Root path returns index.html
	statusCode, body := makeReq("/")
	if statusCode != http.StatusOK {
		t.Errorf("GET / expected 200, got %d", statusCode)
	}
	if !bytes.Contains([]byte(body), []byte("test content")) {
		t.Errorf("GET / body missing 'test content': %s", body)
	}

	// Test 2: SPA routing — unknown path falls back to index.html
	statusCode, body = makeReq("/some/spa/route")
	if statusCode != http.StatusOK {
		t.Errorf("GET /some/spa/route expected 200, got %d", statusCode)
	}
	if !bytes.Contains([]byte(body), []byte("test content")) {
		t.Errorf("GET /some/spa/route body missing 'test content': %s", body)
	}
}
