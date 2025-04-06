package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a Python code execution environment with persistence
type Session struct {
	ID         string
	sessionDir string
	statePath  string
	lastUsed   time.Time
	mutex      sync.Mutex
	isRunning  bool
}

// Manager handles the creation and management of interpreter sessions
type Manager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
	baseDir  string
}

// NewManager creates a new session manager
func NewManager() *Manager {
	// Create a base directory for all sessions
	baseDir := filepath.Join(os.TempDir(), "python-sessions")
	os.MkdirAll(baseDir, 0755)

	return &Manager{
		sessions: make(map[string]*Session),
		baseDir:  baseDir,
	}
}

// GetOrCreateSession retrieves an existing session or creates a new one
func (m *Manager) GetOrCreateSession(id string) (*Session, error) {
	// If ID is provided, try to get existing session
	if id != "" {
		m.mutex.RLock()
		session, exists := m.sessions[id]
		m.mutex.RUnlock()

		if exists && session.isRunning {
			return session, nil
		}
	}

	// Create a new session with the provided ID (or generate one if empty)
	return m.createNewSession(id)
}

// createNewSession initializes a new Python session
func (m *Manager) createNewSession(providedID string) (*Session, error) {
	sessionID := providedID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Create a directory for this session
	sessionDir := filepath.Join(m.baseDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %v", err)
	}

	// Create a state file path for this session
	statePath := filepath.Join(sessionDir, "session_state.py")

	// Create the initial state file
	if err := os.WriteFile(statePath, []byte("# Python session state file\n"), 0644); err != nil {
		return nil, fmt.Errorf("failed to initialize session state: %v", err)
	}

	session := &Session{
		ID:         sessionID,
		sessionDir: sessionDir,
		statePath:  statePath,
		lastUsed:   time.Now(),
		isRunning:  true,
	}

	m.mutex.Lock()
	m.sessions[sessionID] = session
	m.mutex.Unlock()

	return session, nil
}

// ExecuteCode runs Python code within the given session
func (s *Session) ExecuteCode(ctx context.Context, code string) (string, string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return "", "", errors.New("session is no longer running")
	}

	// Update last used time
	s.lastUsed = time.Now()

	// Create a temporary script file that imports the session state
	tempScriptPath := filepath.Join(s.sessionDir, fmt.Sprintf("exec_%d.py", time.Now().UnixNano()))
	scriptContent := fmt.Sprintf(`
# Import the session state
try:
    exec(open(%q).read())
except Exception as e:
    pass  # Ignore errors when loading state

# Execute the provided code
%s

# Save important variables to session state
import inspect, sys
with open(%q, "w") as state_file:
    state_file.write("# Python session state file\n")
    for name, value in list(locals().items()):
        if not name.startswith("_") and name != "state_file" and not inspect.ismodule(value):
            try:
                state_file.write("{} = {!r}\n".format(name, value))
            except:
                pass
`, s.statePath, code, s.statePath)

	if err := os.WriteFile(tempScriptPath, []byte(scriptContent), 0644); err != nil {
		return "", "", fmt.Errorf("failed to create execution script: %v", err)
	}

	// Ensure we clean up the temporary script after execution
	defer os.Remove(tempScriptPath)

	// Execute the script
	cmd := exec.CommandContext(ctx, "python3", tempScriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Special handling for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", "", ctx.Err()
	}

	return stdout.String(), stderr.String(), err
}

// CleanupSession terminates the session and removes its files
func (s *Session) Cleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		s.isRunning = false
		// Remove the session directory
		os.RemoveAll(s.sessionDir)
	}
}

// CleanupSessions removes old sessions
func (m *Manager) CleanupSessions(maxAge time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.lastUsed) > maxAge {
			session.Cleanup()
			delete(m.sessions, id)
		}
	}
}

// GetSessionCount returns the current sessions (for testing purposes)
func (m *Manager) GetSessionCount() map[string]*Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy of the sessions map
	sessionsCopy := make(map[string]*Session)
	for id, session := range m.sessions {
		sessionsCopy[id] = session
	}
	return sessionsCopy
}
