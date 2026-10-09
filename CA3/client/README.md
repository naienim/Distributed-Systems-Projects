# Client README

This directory contains the command-line client for the replicated key-value store.

The client is used to send PUT, GET, and dump requests to the replica servers.

All commands should be executed from the root directory of the project.

## General Command Format

```bash
go run client/main.go -op <operation> -replica <replica_url> -key <key> -value <value> -mode <mode>
```

## Supported Operations

The client supports the following operations:

put
get
dump

## Supported Consistency Modes

The client supports the following consistency modes:

eventual
strong

## PUT in Eventual Consistency Mode

Example:

```bash
go run client/main.go -op put -replica http://localhost:8001 -key x -value 10 -mode eventual
```

This command writes x=10 to Replica 1 in eventual consistency mode.

In this mode, the receiving replica stores the value locally and returns a response immediately. Replication to other replicas happens asynchronously.

## GET in Eventual Consistency Mode

Example:

```bash
go run client/main.go -op get -replica http://localhost:8002 -key x -mode eventual
```

This command reads key x from Replica 2 in eventual consistency mode.

In this mode, the selected replica reads only from its local data. Therefore, a stale read or key not found may happen if replication has not completed yet.

## PUT in Strong Consistency Mode

Example:

```bash
go run client/main.go -op put -replica http://localhost:8001 -key y -value 20 -mode strong
```

This command writes y=20 to Replica 1 in strong consistency mode.

In this mode, the PUT operation succeeds only if a majority of replicas acknowledge the update.

Since the system has three replicas, at least two replicas must accept the update.

## GET in Strong Consistency Mode

Example:

```bash
go run client/main.go -op get -replica http://localhost:8002 -key y -mode strong
```

This command reads key y from Replica 2 in strong consistency mode.

In this mode, the replica checks other replicas and returns the newest available version.

## Dump Local Data

Example:

```bash
go run client/main.go -op dump -replica http://localhost:8001
```

This command prints all local data stored in Replica 1.

## Example Workflow

Start all three replicas first.

Then run:

```bash
go run client/main.go -op put -replica http://localhost:8001 -key x -value 10 -mode eventual
```

Immediately read from Replica 2:

```bash
go run client/main.go -op get -replica http://localhost:8002 -key x -mode eventual
```

If replication delay is high, this may return:

```text
key not found
```

After a few seconds, run the same GET command again:

```bash
go run client/main.go -op get -replica http://localhost:8002 -key x -mode eventual
```

Now the value should be visible.

## Useful Commands

Read from Replica 1:

```bash
go run client/main.go -op get -replica http://localhost:8001 -key x -mode eventual
```

Read from Replica 2:

```bash
go run client/main.go -op get -replica http://localhost:8002 -key x -mode eventual
```

Read from Replica 3:

```bash
go run client/main.go -op get -replica http://localhost:8003 -key x -mode eventual
```

Dump Replica 1:

```bash
go run client/main.go -op dump -replica http://localhost:8001
```

Dump Replica 2:

```bash
go run client/main.go -op dump -replica http://localhost:8002
```

Dump Replica 3:

```bash
go run client/main.go -op dump -replica http://localhost:8003
```

## Client Output

The client prints the HTTP status, measured latency, and JSON response.

Important output fields:

ok:
Shows whether the operation succeeded.

message:
Explains the result of the operation.

record:
The returned or written record.

updated_replicas:
Number of replicas updated during PUT.

checked_replicas:
Number of replicas checked during GET.

latency_ms:
Latency reported by the replica server.

Client measured latency:
Latency measured by the client program.

## Common Errors

Connection refused:
This means the selected replica is not running. Start the replica and run the command again.

Key not found:
This means the selected replica does not currently have the requested key. In eventual consistency mode, this may happen temporarily before replication completes.

Majority not available:
This may happen in strong consistency mode when fewer than two replicas are available.

Example message:

```text
PUT failed: majority of replicas is not available
```

Start more replicas and try again.