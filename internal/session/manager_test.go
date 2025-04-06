package session

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.sessions == nil {
		t.Fatal("Expected non-nil sessions map")
	}

	if manager.baseDir == "" {
		t.Fatal("Expected non-empty baseDir")
	}

	// Verify the base directory exists
	_, err := os.Stat(manager.baseDir)
	if os.IsNotExist(err) {
		t.Fatalf("Base directory was not created: %s", manager.baseDir)
	}
}

func TestGetOrCreateSession(t *testing.T) {
	manager := NewManager()

	// Test creating a new session
	session1, err := manager.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("Failed to create new session: %v", err)
	}

	if session1.ID == "" {
		t.Fatal("Expected session to have a non-empty ID")
	}

	if !session1.isRunning {
		t.Fatal("Expected new session to be running")
	}

	// Test getting an existing session
	session2, err := manager.GetOrCreateSession(session1.ID)
	if err != nil {
		t.Fatalf("Failed to get existing session: %v", err)
	}

	if session1.ID != session2.ID {
		t.Fatalf("Expected to get the same session with ID %s, but got %s", session1.ID, session2.ID)
	}

	// Verify session directory structure
	if _, err := os.Stat(session1.sessionDir); os.IsNotExist(err) {
		t.Fatalf("Session directory was not created: %s", session1.sessionDir)
	}

	if _, err := os.Stat(session1.statePath); os.IsNotExist(err) {
		t.Fatalf("Session state file was not created: %s", session1.statePath)
	}
}

func TestExecuteCode(t *testing.T) {
	manager := NewManager()
	session, err := manager.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test simple code execution
	stdout, stderr, err := session.ExecuteCode(context.Background(), "print('Hello, World!')")
	if err != nil {
		t.Fatalf("Failed to execute code: %v", err)
	}

	if stdout != "Hello, World!\n" {
		t.Fatalf("Expected stdout 'Hello, World!\\n', got '%s'", stdout)
	}

	if stderr != "" {
		t.Fatalf("Expected empty stderr, got '%s'", stderr)
	}

	// Test with code that should cause an error
	stdout, stderr, err = session.ExecuteCode(context.Background(), "print(undefined_variable)")
	if err == nil {
		t.Fatal("Expected error when executing invalid code")
	}

	if stderr == "" {
		t.Fatal("Expected non-empty stderr for invalid code")
	}

	// Test state persistence
	_, _, err = session.ExecuteCode(context.Background(), "x = 42")
	if err != nil {
		t.Fatalf("Failed to set variable: %v", err)
	}

	stdout, _, err = session.ExecuteCode(context.Background(), "print(x)")
	if err != nil {
		t.Fatalf("Failed to read variable: %v", err)
	}

	if stdout != "42\n" {
		t.Fatalf("Expected stdout '42\\n', got '%s'", stdout)
	}

	// Test with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err = session.ExecuteCode(ctx, "import time; time.sleep(1)")
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("Expected deadline exceeded error, got: %v", err)
	}
}

func TestCleanupSessions(t *testing.T) {
	manager := NewManager()

	// Create a session
	session, err := manager.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionDir := session.sessionDir

	// Make the session appear old
	session.lastUsed = time.Now().Add(-2 * time.Hour)

	// Run cleanup with 1 hour max age
	manager.CleanupSessions(1 * time.Hour)

	// Session should be removed
	if _, exists := manager.sessions[session.ID]; exists {
		t.Fatal("Session should have been removed")
	}

	// Directory should be removed
	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Fatal("Session directory should have been removed")
	}
}

func TestSessionCleanup(t *testing.T) {
	manager := NewManager()
	session, err := manager.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionDir := session.sessionDir

	// Clean up the session manually
	session.Cleanup()

	if session.isRunning {
		t.Fatal("Session should not be running after cleanup")
	}

	// Directory should be removed
	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Fatal("Session directory should have been removed")
	}
}

func TestConcurrentSessionUsage(t *testing.T) {
	manager := NewManager()
	session, err := manager.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Run multiple executions concurrently
	const concurrentRuns = 10
	errChan := make(chan error, concurrentRuns)

	for i := 0; i < concurrentRuns; i++ {
		go func(i int) {
			code := fmt.Sprintf("print('Concurrent execution %d')", i)
			_, _, err := session.ExecuteCode(context.Background(), code)
			errChan <- err
		}(i)
	}

	// Collect all errors
	for i := 0; i < concurrentRuns; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("Concurrent execution failed: %v", err)
		}
	}
}
