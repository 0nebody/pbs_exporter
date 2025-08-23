package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestRedirectToMetrics(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	}
	w := httptest.NewRecorder()
	redirectToMetrics(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Errorf("Expected status code %d, got %d", http.StatusFound, res.StatusCode)
	}

	location, err := res.Location()
	if err != nil {
		t.Fatalf("Failed to get Location header: %v", err)
	}
	if location.Path != "/metrics" {
		t.Errorf("Expected Location header to be '/metrics', got '%s'", location.Path)
	}
}

func TestHealthz(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/healthz"},
	}
	w := httptest.NewRecorder()
	healthz(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}
}

func TestMain(t *testing.T) {
	listenPort := 9999
	if _, err := os.Stat("../../pbs_exporter"); os.IsNotExist(err) {
		t.Skip("pbs_exporter command not found, skipping tests")
	}

	var stderr bytes.Buffer
	command := exec.Command("../../pbs_exporter", fmt.Sprintf("--web.listen-address=:%d", listenPort), "--no-job.enabled")
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("Failed to start pbs_exporter: %v", err)
	}
	defer command.Process.Kill()

	// wait for web server to start
	const maxWait = 5 * time.Second
	const pollInterval = 100 * time.Millisecond
	startTime := time.Now()
	for {
		if time.Since(startTime) > maxWait {
			t.Fatalf("Server timed out after %v. Stderr: %s", maxWait, stderr.String())
		}
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", listenPort))
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(pollInterval)
	}

	t.Run("TestRedirectToMetrics", func(t *testing.T) {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", listenPort))
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Errorf("Expected status code %d, got %d", http.StatusFound, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/metrics" {
			t.Errorf("Expected Location header to be '/metrics', got '%s'", loc)
		}
	})

	t.Run("TestHealthz", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", listenPort))
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("TestMetrics", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", listenPort))
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		if len(body) == 0 {
			t.Error("Expected non-empty metrics response, got empty body")
		}
	})
}
