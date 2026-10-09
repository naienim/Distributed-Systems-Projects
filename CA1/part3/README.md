# Part 3: HTTP Service, Docker Containerization, and VM Execution

## Overview

This project implements a simple HTTP calculator service in Go. The service is containerized using Docker and is designed to run inside a virtual machine. The host machine can send HTTP requests to the service running inside the VM.

The service has two required endpoints:

- `GET /health`
- `GET /compute`

The `/health` endpoint checks whether the service is running.

The `/compute` endpoint receives a computation request using query parameters and returns the result as JSON.

Example:

```bash
curl "http://localhost:8080/compute?operator=add&a=10&b=20"
```

Response:

```json
{
  "operation": "add",
  "a": 10,
  "b": 20,
  "result": 30
}
```

## Project Structure

```text
part3/
├── main.go
├── Dockerfile
└── README.md
```

## Requirements

Main packages used:

```text
net/http
encoding/json
strconv
strings
log
time
math
```

## Supported Endpoints

### Health Endpoint

```http
GET /health
```

Example:

```bash
curl http://localhost:8080/health
```

Example response:

```json
{
  "status": "ok",
  "service": "distributed-systems-hw1-part3",
  "time": "2026-05-09T12:40:00+03:00"
}
```

### Compute Endpoint

```http
GET /compute?op=<operation>&a=<number>&b=<number>
```

Example:

```bash
curl "http://localhost:8080/compute?operator=add&a=10&b=20"
```

Example response:

```json
{
  "operation": "add",
  "a": 10,
  "b": 20,
  "result": 30
}
```

## Supported Operations

The service supports:

```text
add
sub
mul
div
mod
pow
```

## Error Handling

The service handles these errors:

```text
missing parameters
invalid operation
non-numeric input
division by zero
wrong HTTP method
```

Examples:

```bash
curl "http://localhost:8080/compute?op=div&a=8&b=0"
```

Response:

```json
{
  "error": "division_by_zero"
}
```

```bash
curl "http://localhost:8080/compute?operator=abc&a=5&b=7"
```

Response:

```json
{
  "error": "invalid_operation"
}
```

## Run Locally Without Docker

```bash
go run main.go
```

Test it:

```bash
curl http://localhost:8080/health
curl "http://localhost:8080/compute?operator=add&a=5&b=7"
```

Stop the service with:

```text
Ctrl + C
```

## Docker Build Command

Build the Docker image inside the `part3` directory:

```bash
sudo docker build -t hw1-part3-service .
```

## Docker Run Command

Run the container:

```bash
sudo docker run -d --name hw1-part3-container -p 8080:8080 hw1-part3-service
```

The port mapping is:

```text
VM port 8080 -> Container port 8080
```

## Port Number

The service uses port:

```text
8080
```

## Check Running Container

Check that the container is running:

```bash
sudo docker ps
```

Output port mapping:

```text
0.0.0.0:8080->8080/tcp
```

Check logs:

```bash
sudo docker logs hw1-part3-container
```

Output:

```text
<date> <time> HTTP service is running on port 8080
```

## Test Inside the VM

After running the container inside the VM, test:

```bash
curl http://localhost:8080/health
```

```bash
curl "http://localhost:8080/compute?operator=add&a=10&b=20"
```

## Connect from Host to VM

Inside the VM, find the VM IP:

```bash
hostname -I
```

Example output:

```text
192.168.1.45
```

From the host machine (windows in my case), run:

```bash
curl http://192.168.1.45:8080/health
```

```bash
curl "http://192.168.1.45:8080/compute?operator=add&a=10&b=20"
```

Replace `192.168.1.45` with the actual VM IP.

## Stop and Remove Container

Stop the container:

```bash
sudo docker stop hw1-part3-container
```

Remove the container:

```bash
sudo docker rm hw1-part3-container
```