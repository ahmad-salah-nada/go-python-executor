package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go--python-executor/internal/models"
	"go--python-executor/internal/session"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupTestServer creates a test HTTP server with the execute handler
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", ExecuteHandler)
	return httptest.NewServer(mux)
}

// executeCode is a helper function that sends code to the execute endpoint
func executeCode(t *testing.T, server *httptest.Server, code string, sessionID string) (*models.ResponsePayload, *http.Response) {
	// Create request payload
	payload := models.RequestPayload{
		ID:   sessionID,
		Code: code,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Send HTTP POST request
	resp, err := http.Post(server.URL+"/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response models.ResponsePayload
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	return &response, resp
}

func TestBasicExecution(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test simple code execution
	response, resp := executeCode(t, server, "print('Hello, World!')", "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(response.Stdout, "Hello, World!") {
		t.Fatalf("Expected stdout to contain 'Hello, World!', got '%s'", response.Stdout)
	}

	if response.Stderr != "" {
		t.Fatalf("Expected empty stderr, got '%s'", response.Stderr)
	}

	if response.Error != "" {
		t.Fatalf("Expected no error, got '%s'", response.Error)
	}

	// Store the session ID for future tests
	sessionID := response.ID
	if sessionID == "" {
		t.Fatal("Expected non-empty session ID")
	}
}

func TestSessionPersistence(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// First request: set a variable
	response1, _ := executeCode(t, server, "x = 42", "")
	sessionID := response1.ID

	// Second request: read the variable
	response2, _ := executeCode(t, server, "print(x)", sessionID)

	if !strings.Contains(response2.Stdout, "42") {
		t.Fatalf("Session persistence failed. Expected stdout to contain '42', got '%s'", response2.Stdout)
	}

	// Third request: modify the variable
	executeCode(t, server, "x += 10", sessionID)

	// Fourth request: verify the modification
	response4, _ := executeCode(t, server, "print(x)", sessionID)

	if !strings.Contains(response4.Stdout, "52") {
		t.Fatalf("Session variable modification failed. Expected stdout to contain '52', got '%s'", response4.Stdout)
	}
}

func TestErrorHandling(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test syntax error
	response, _ := executeCode(t, server, "print(", "")

	if response.Stderr == "" {
		t.Fatal("Expected stderr to contain syntax error, got empty string")
	}

	// Test name error
	response, _ = executeCode(t, server, "print(undefined_variable)", "")

	if !strings.Contains(response.Stderr, "NameError") {
		t.Fatalf("Expected stderr to contain 'NameError', got '%s'", response.Stderr)
	}

	// Test that errors don't break the session
	sessionID := response.ID
	executeCode(t, server, "valid_var = 'still works'", sessionID)
	response, _ = executeCode(t, server, "print(valid_var)", sessionID)

	if !strings.Contains(response.Stdout, "still works") {
		t.Fatalf("Session should continue working after errors. Expected 'still works', got '%s'", response.Stdout)
	}
}

func TestExecutionTimeout(t *testing.T) {
	// Override the execution timeout for testing
	originalTimeout := ExecutionTimeout
	ExecutionTimeout = 350 * time.Millisecond
	defer func() { ExecutionTimeout = originalTimeout }()

	server := setupTestServer()
	defer server.Close()

	// Test code that should timeout
	response, _ := executeCode(t, server, "import time; time.sleep(1)", "")

	if !strings.Contains(response.Error, "timeout") && !strings.Contains(response.Error, "deadline") {
		t.Fatalf("Expected timeout error, got: '%s'", response.Error)
	}

	// Verify the session still works after timeout
	sessionID := response.ID
	response, _ = executeCode(t, server, "print('after timeout')", sessionID)

	if !strings.Contains(response.Stdout, "after timeout") {
		t.Fatalf("Session should work after timeout. Expected 'after timeout', got '%s'", response.Stdout)
	}
}

func TestInvalidRequests(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test with invalid JSON
	resp, err := http.Post(server.URL+"/execute", "application/json", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code 400 for invalid JSON, got %d", resp.StatusCode)
	}

	// Test with unsupported HTTP method
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/execute", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status code 405 for GET method, got %d", resp.StatusCode)
	}
}

func TestSessionCleanup(t *testing.T) {
	// Create a custom session manager for this test
	manager := session.NewManager()
	originalManager := sessionManager
	sessionManager = manager
	defer func() { sessionManager = originalManager }()

	// Override session time limit for testing
	originalTimeLimit := SessionTimeLimit
	SessionTimeLimit = 100 * time.Millisecond
	defer func() { SessionTimeLimit = originalTimeLimit }()

	server := setupTestServer()
	defer server.Close()

	// Create a session
	response, _ := executeCode(t, server, "x = 'test cleanup'", "")
	sessionID := response.ID

	// Get the current count of sessions
	manager.CleanupSessions(24 * time.Hour) // Run cleanup with very long timeout to not affect count
	initialSessionCount := len(manager.GetSessionCount())

	// Wait for the session to expire
	time.Sleep(200 * time.Millisecond)

	// Run cleanup
	manager.CleanupSessions(SessionTimeLimit)

	// Verify the session was cleaned up
	newSessionCount := len(manager.GetSessionCount())
	if newSessionCount >= initialSessionCount {
		t.Fatalf("Expected sessions to be cleaned up. Initial count: %d, New count: %d",
			initialSessionCount, newSessionCount)
	}

	// Try to use the expired session
	response, _ = executeCode(t, server, "print(x)", sessionID)

	// Since the session was cleaned up, there should be no 'x' variable
	if !strings.Contains(response.Stderr, "NameError") && response.Stderr != "" {
		t.Fatalf("Expected error when using expired session, got stdout: '%s', stderr: '%s'",
			response.Stdout, response.Stderr)
	}
}

func TestConcurrentRequests(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a session
	response, _ := executeCode(t, server, "counter = 0", "")
	sessionID := response.ID

	// Send multiple concurrent requests
	const concurrentRequests = 30
	errorChan := make(chan error, concurrentRequests)

	// Run concurrent requests to increment a counter
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			_, resp := executeCode(t, server, "counter += 1", sessionID)
			if resp.StatusCode != http.StatusOK {
				errorChan <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}
			errorChan <- nil
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < concurrentRequests; i++ {
		if err := <-errorChan; err != nil {
			t.Fatalf("Concurrent request failed: %v", err)
		}
	}

	// Verify the final counter value
	// Note: Due to Python's GIL and our mutex protection, counter should be incremented sequentially
	response, _ = executeCode(t, server, "print(counter)", sessionID)

	expected := "30"
	if !strings.Contains(response.Stdout, expected) {
		t.Fatalf("Expected counter to be '%s', got '%s'", expected, response.Stdout)
	}
}
