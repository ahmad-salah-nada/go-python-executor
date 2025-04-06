package handler

import (
	"context"
	"encoding/json"
	"go--python-executor/internal/models"
	"go--python-executor/internal/session"
	"net/http"
	"sync"
	"time"
)

var (
	ExecutionTimeout = 2 * time.Second  // 2-second execution limit
	SessionTimeLimit = 5 * time.Minute  // 5-minute session lifetime
	CleanupInterval  = 30 * time.Second // Cleanup old sessions every minute
)

var (
	sessionManager *session.Manager
	once           sync.Once
)

// getSessionManager returns the singleton session manager
func getSessionManager() *session.Manager {
	once.Do(func() {
		sessionManager = session.NewManager()

		// Start a goroutine to clean up old sessions
		go func() {
			for {
				time.Sleep(CleanupInterval)
				sessionManager.CleanupSessions(SessionTimeLimit)
			}
		}()
	})
	return sessionManager
}

// Helper function to send error responses
func sendErrorResponse(w http.ResponseWriter, sessionID, message string) {
	w.Header().Set("Content-Type", "application/json")
	response := models.ResponsePayload{
		ID:    sessionID,
		Error: message,
	}
	json.NewEncoder(w).Encode(response)
}

// ExecuteHandler processes Python code execution requests
func ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req models.RequestPayload
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), ExecutionTimeout)
	defer cancel()

	// Get or create session
	manager := getSessionManager()
	session, err := manager.GetOrCreateSession(req.ID)
	if err != nil {
		sendErrorResponse(w, "", "Failed to initialize session")
		return
	}

	// Execute code in the session
	stdout, stderr, err := session.ExecuteCode(ctx, req.Code)

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		sendErrorResponse(w, session.ID, "execution timeout")
		return
	}

	// Prepare response
	response := models.ResponsePayload{
		ID:     session.ID,
		Stdout: stdout,
		Stderr: stderr,
	}

	// Handle errors
	if err != nil && stderr == "" {
		response.Error = err.Error()
		response.Stdout = ""
		response.Stderr = ""
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
