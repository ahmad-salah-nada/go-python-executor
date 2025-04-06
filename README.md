# Go-Python Executor

A web service that allows execution of Python code. Built with Go and designed to maintain persistent Python sessions.

## Features

- **Python Code Execution**: Execute Python code via HTTP API endpoints
- **Session Management**: Maintain stateful Python sessions for code that builds upon previous executions
- **Timeout Handling**: Configurable execution timeouts
- **Concurrency Support**: Handles multiple concurrent requests efficiently
- **Docker Deployment**: Ready to deploy with Docker and docker-compose
- **HTTPS Support**: Configured with Caddy for automatic HTTPS

## Architecture

The project consists of:

- Go backend service that handles HTTP requests and manages Python sessions
- Python interpreter that executes the submitted code
- Caddy reverse proxy for HTTPS termination

## Prerequisites

- Go 1.24 or later
- Docker and docker-compose (for containerized deployment)
- Python 3.x

## Installation

### Clone the repository

```bash
git clone https://github.com/yourusername/go-python-executor.git
cd go-python-executor
```

### Local Development Setup

1. Install dependencies:

```bash
go mod tidy
```

2. Build and run:

```bash
go build -o server ./cmd/server/main.go
./server
```

### Docker Deployment

1. Build and start the containers:

```bash
docker-compose up -d
```

This will start both the Go server and the Caddy reverse proxy.

## API Usage

### Execute Python Code

**Endpoint**: `POST /execute`

**Request Body**:

```json
{
  "id": "optional-session-id",
  "code": "print('Hello, World!')"
}
```

- `id`: (Optional) Session ID for continuing a previous execution. If not provided, a new session will be created.
- `code`: Python code to be executed.

**Response**:

```json
{
  "id": "session-id",
  "stdout": "Hello, World!",
  "stderr": "",
  "error": ""
}
```

- `id`: Session ID that can be used for subsequent requests
- `stdout`: Standard output from the executed code
- `stderr`: Standard error output
- `error`: Any execution errors or timeouts

## Testing

Run the test suite:

```bash
go test ./...
```