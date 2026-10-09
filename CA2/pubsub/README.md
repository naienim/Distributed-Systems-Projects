# Pub/Sub VM — Memory Monitoring Publisher and Subscriber

This folder contains the code used for the Subscribe/Publish part of the assignment.

The design is simple:

- The web service on VM1 acts as the publisher
- A separate subscriber program listens for alerts
- When VM1 memory usage becomes too high, the publisher sends an event over TCP
- The subscriber shows a clear alert message
- The publisher also works as a JSON-RPC client for VM2 authentication and VM3 file retrieval

## Files

- publisher.go: web server, authentication RPC client, file RPC client, memory monitor and event publisher
- subscriber.go: TCP subscriber that receives alert events
- templates/: HTML templates used by the publisher
- static/: CSS and other static assets served by the publisher at /static/

## Requirements

- Go installed on Linux
- Ports 8080 and 7000 open on the relevant machines
- The publisher and subscriber must use real network addresses, not localhost
- Update the hardcoded IP addresses before running
- Port 8081 open on VM2 for authentication RPC
- Port 9090 open on VM3 for file RPC

## Configuration

Edit these constants in publisher.go:

```go
const (
    authHost = "AUTH_VM_IP"
    authPort = "8081"

    fileHost = "FILE_VM_IP"
    filePort = "9090"

    subHost = "SUBSCRIBER_VM_IP"
    subPort = "7000"

    thresholdMB = 300
    monitorIntervalSecond = 3
)
```

## Run the subscriber

From inside the pubsub/ directory:

```bash
go run subscriber.go
```

The subscriber waits on port 7000

When an alert arrives, it shows a message like:

```text
ALERT: HIGH MEMORY USAGE RECEIVED
```

## Run the publisher/web server

From inside the pubsub/ directory:

```bash
go run publisher.go
```

The publisher starts the web interface on port 8080.

## Web endpoints

### GET /
Displays the login page.

### POST /login
Authenticates the user by calling VM2 via JSON-RPC.

After a successful login, the publisher also calls VM3 via JSON-RPC to retrieve the sample text file and image.

### GET /consume-memory?mb=100
Allocates memory and keeps it referenced so the memory usage increases.

Example:

```bash
curl "http://WEB_VM_IP:8080/consume-memory?mb=100"
```

## Event format

The publisher sends a JSON event similar to this:

```json
{
  "event_type": "HIGH_MEMORY_USAGE",
  "service": "web-server",
  "memory_mb": 345,
  "threshold_mb": 300,
  "timestamp": "2026-05-26T12:00:00Z"
}
```

## VM3 file retrieval

In the Pub/Sub version, the publisher does not load files from VM3 using HTTP URLs. Instead, it calls the RPC method on VM3:

```text
FileService.GetFile
```

The publisher requests:

```text
files/sample.txt
images/sample.png
```

VM3 returns the file content as base64 encoded data. The publisher decodes the text file and displays it in the welcome page. For the image, the publisher builds a base64 data URI and passes it to the HTML template.

Because Go’s html/template may sanitize normal strings inside an image src attribute, the image data is passed as template.URL.

## Memory monitoring method

The publisher periodically reads process memory statistics using runtime.ReadMemStats(). It checks the heap allocation and publishes an event when the value is above the configured threshold.

## Deliberately increasing memory usage

The /consume-memory endpoint allocates a byte slice of the requested size and stores it in a global slice:

```go
var allocatedBlocks [][]byte
```

This keeps the memory in use so the threshold can be reached during testing.

## Testing sequence

1. Start the authentication service on VM2.
2. Start the file RPC service on VM3.
3. Start subscriber.go on the subscriber machine.
4. Start publisher.go on VM1.
5. Open the login page.
6. Log in with valid credentials.
7. Confirm that the welcome page displays the text file and image received from VM3 through RPC.
8. Call /consume-memory one or more times.
9. Watch the subscriber terminal for the alert.