# Web VM (VM1) — Web Interface and RPC Client

This service is the user-facing web server for the distributed system. It shows the login page, sends authentication requests to the authentication service on VM2 using JSON-RPC and after a successful login retrieves a file and image from VM3 using JSON-RPC.

## Responsibilities

- Serve the login page
- Collect username and password from the user
- Call the authentication service on VM2 through RPC
- Show an error message for failed logins
- Show a welcome page for successful logins
- Retrieve a file and an image from VM3 through RPC
- Display the received file and image on the welcome page
- Provide a /consume-memory endpoint for memory stress testing (implemented by the publisher service in pubsub/publisher.go)
- Publish high-memory events to the subscriber in the Pub/Sub part (implemented by the publisher service in pubsub/publisher.go)

## Files

- main.go: web server, login handler, authentication RPC client and file RPC client
- templates/login.html: login page
- templates/welcome.html: success page
- static/style.css: optional CSS file

## Requirements

- Go installed on Linux
- Network access to VM2 and VM3
- Port 8080 open on VM1
- Port 8081 open on VM2 for authentication RPC
- Port 9090 open on VM3 for file RPC
- Replace the hardcoded VM IP addresses in main.go with the real VM addresses before running

## Configuration

Edit the constants in main.go:

```go
const (
    authHost = "AUTH_VM_IP"
    authPort = "8081"

    fileHost = "FILE_VM_IP"
    filePort = "9090"
)
```

For the Pub/Sub version, also update:

```go
subHost = "SUBSCRIBER_VM_IP"
subPort = "7000"
```

## Run

From inside the web-vm/ directory:

```bash
go run main.go
```

The service starts on port 8080.

## Test the login page

Open the browser and visit the VM1 address:

```text
http://WEB_VM_IP:8080
```

Try these sample users from VM2:

- alice (Password: alice123)
- bob (Password: bob123)
- admin (Password: admin123)

The passwords are entered normally by the user, but VM2 stores and compares their SHA-256 hashes.

## Endpoints

### GET /
Shows the login page.

Sends an RPC request to VM2 and shows:

- an error message if authentication fails
- the welcome page if authentication succeeds

After authentication succeeds, VM1 also sends RPC requests to VM3 to retrieve the sample text file and image.

### GET /consume-memory?mb=100 (implemented in pubsub/publisher.go)
Allocates the requested amount of memory in MB and keeps it referenced in memory so it is not freed immediately.

Example:

```bash
curl "http://WEB_VM_IP:8080/consume-memory?mb=100"
```

## How VM3 content is displayed

After a successful login, the web server calls VM3 using JSON-RPC. It no longer loads VM3 files using HTTP URLs.

VM1 calls the following RPC method on VM3:

```text
FileService.GetFile
```

## Pub/Sub behavior

The web service in web-vm/main.go only handles the normal web and RPC behavior. The Pub/Sub behavior is implemented in pubsub/publisher.go.

In the Pub/Sub version, the publisher periodically checks its memory usage using Go’s runtime.ReadMemStats. When memory usage passes the threshold, it publishes a JSON event over TCP to the subscriber.

## Testing notes

1. Start the authentication service on VM2.
2. Start the file RPC service on VM3.
3. Start the web service on VM1.
4. Open the login page.
5. Log in with valid credentials.
6. Confirm that the text file and image are displayed through RPC.
7. Use /consume-memory several times (Pub/Sub version).
8. Check the subscriber terminal for the alert (Pub/Sub version).