# Authentication VM (VM2) — JSON-RPC Login Service

This service stores user data locally on VM2 and exposes a JSON-RPC login verification procedure. VM1 calls this service during the login process.

## Responsibilities

- Store user data on VM2
- Store password hashes instead of plain text passwords
- Provide an RPC method for login verification
- Hash the received password and compare it with the stored password hash


## Files

- main.go: authentication RPC server
- users.json: sample user database with hashed passwords

## Sample users

The service includes at least three test users:

- alice (Password: alice123)
- bob (Password: bob123)
- admin (Password: admin123)

The users still log in with the passwords shown above, but the actual users.json file stores SHA-256 password hashes instead of plain text passwords.

## Requirements

- Go installed on Linux
- Port 8081 open on VM2
- users.json must be in the same directory as main.go
- The web server must point to the correct VM2 IP address

## RPC method

The exposed RPC method is:

```text
AuthenticationService.Login
```

Request fields:

- username
- password

Response fields:

- success
- message

## Run

From inside the auth-vm/ directory:

```bash
go run main.go
```

The service listens on port 8081

## Testing

The normal way to test this service is through VM1’s web page.

- wrong password: show an error
- correct credentials: show the welcome page

## Data file format

users.json uses a JSON array. Passwords are stored as SHA-256 hashes:

```json
[
  {
    "username": "alice",
    "password_hash": "4e40e8ffe0ee32fa53e139147ed559229a5930f89c2204706fc174beb36210b3"
  },
  {
    "username": "bob",
    "password_hash": "8d059c3640b97180dd2ee453e20d34ab0cb0f2eccbe87d01915a8e578a202b11"
  },
  {
    "username": "admin",
    "password_hash": "240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9"
  }
]
