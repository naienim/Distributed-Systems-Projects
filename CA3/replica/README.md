# Replica Server README

This directory contains the implementation of the replica server for the replicated key-value store.

Each replica is an independent HTTP server. The system uses three replicas, and each replica must be started as a separate process with its own configuration file. The replicas communicate with each other over HTTP.

## Replica Ports

Replica 1: http://localhost:8001
Replica 2: http://localhost:8002
Replica 3: http://localhost:8003

## Configuration Files

Each replica is started using a JSON configuration file located in the configs directory.

Example configuration file for Replica 1:

```json
{
  "id": "replica1",
  "port": 8001,
  "peers": [
    "http://localhost:8002",
    "http://localhost:8003"
  ],
  "replication_delay_ms": 500,
  "data_file": "results/replica1_data.json"
}
```

## Configuration Fields

id:
The unique ID of the replica.

port:
The port number used by the HTTP server.

peers:
The list of other replicas.

replication_delay_ms:
Artificial replication delay in milliseconds. This is used to simulate network delay.

data_file:
The local JSON file used by the replica to store its data.

## How to Run the Replicas

Run all commands from the root directory of the project.

Open three separate terminals.

Terminal 1:

```bash
go run replica/main.go -config configs/replica1.json
```

Terminal 2:

```bash
go run replica/main.go -config configs/replica2.json
```

Terminal 3:

```bash
go run replica/main.go -config configs/replica3.json
```

After starting all three servers, the system is ready to receive requests from the client.

## Available HTTP Endpoints

PUT endpoint:

```text
POST /put
```

This endpoint is used by the client to write or update a key-value pair.

GET endpoint:

```text
GET /get?key=<key>&mode=<mode>
```

Example:

```text
http://localhost:8001/get?key=x&mode=eventual
```

DUMP endpoint:

```text
GET /dump
```

This endpoint returns all local data stored in the selected replica.

Example:

```text
http://localhost:8001/dump
```

Internal replication endpoint:

```text
POST /internal/replicate
```

This endpoint is used internally by replicas to send replication messages to each other.

Internal GET endpoint:

```text
GET /internal/get?key=<key>
```

This endpoint is used internally during coordinated strong reads.

## Important Note About Browser Access

The root path / is not implemented. Therefore, opening the following URL in a browser may return 404 page not found:

```text
http://localhost:8001
```

This is normal.

Use /dump instead:

```text
http://localhost:8001/dump
```

## Consistency Modes

The replica supports two consistency modes:

eventual
strong

## Eventual Consistency

In eventual consistency mode, the receiving replica applies the PUT operation locally and immediately returns a successful response to the client. Then it sends the update to other replicas asynchronously.

This mode has lower PUT latency, but temporary inconsistency and stale reads may happen.

## Strong Consistency

In strong consistency mode, a PUT operation succeeds only when a majority of replicas acknowledge the update.

Since the system has three replicas, the majority is two replicas.

If the majority is not available, the PUT operation fails.

## Versioning

Each record has a version number. The version is increased after every successful PUT operation.

Example record:

```json
{
  "key": "x",
  "value": "10",
  "version": 1,
  "updated_by": "replica1",
  "updated_at": 1781946204655547000
}
```

Versioning prevents older updates from overwriting newer values.

## Conflict Resolution

If two records have the same version but different values, the system resolves the conflict using the following policy:

1. The record with the higher version wins.
2. If versions are equal, the record with the larger updated_by replica ID wins.
3. If both version and updated_by are equal, the newer timestamp wins.

Replica priority:

```text
replica3 > replica2 > replica1
```

## Cleaning Replica Data

Before running a new experiment, stop all replicas and delete old data files.

Windows PowerShell:

```powershell
Remove-Item results\*.json -ErrorAction SilentlyContinue
```

Linux or macOS:

```bash
rm -f results/*.json
```

Then restart all replicas.

## Common Problems

If the browser shows 404 page not found:
This is normal for the root path. Use /dump instead.

If a port is already in use:
Stop the previous replica process or close the terminal running it.

If a replica returns connection refused:
The target replica is not running. Start that replica again.

If old data appears in a new experiment:
Stop all replicas, delete results/*.json, and restart the replicas.