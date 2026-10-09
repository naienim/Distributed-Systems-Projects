# Part 1 – IPC Between Two Processes Using Named Pipes

## Overview

This part implements two independent Go programs that communicate via named pipes (FIFOs):

- worker: a computing service.
- interface: a client that sends requests to the worker and prints responses.

Communication uses a simple text request format and JSON responses.

## Files

- worker.go: – Worker process implementation.
- interface.go: – Interface process implementation.
- README.md – This documentation.
- Named pipes created under /temp:
    - Request FIFO: /temp/request.fifo
    - Response FIFO: /temp/response.fifo

## Build Instructions

From the part1 directory:

```bash
go build -o worker worker.go
go build -o interface interface.go
```

This creates two binaries: worker and interface.

## Run Instructions

1. Start the worker first:

    ```bash
    ./worker
    ```

    The worker:
    - Creates the FIFOs if they do not exist.
    - Opens temp/request.fifo for reading.
    - Opens /temp/response.fifo for writing.
    - Waits for requests and responds.

2. In a separate terminal, start the interface:

    ```bash
    ./interface
    ```

    The interface:
    - Waits for the FIFOs.
    - Opens /temp/request.fifo for writing.
    - Opens /temp/response.fifo for reading.
    - Reads commands and prints responses.

## Protocol

### Request Format

Each request is a single line:

```text
OP A B
```

Where:

- OP is one of the {ADD, SUB, MUL, DIV, MOD, POW, MAX, MIN}
- A and B are numeric operands.

Examples:

```text
ADD 5 8
SUB 12 3
MUL 3 6
DIV 8 6
MOD 8 7
POW 6 12
MAX 12 7
MIN 7 6
```

### Response Format

The worker responds in JSON:

```json
{
    "status": "OK" | "ERROR",
    "result": <number>,   // present only on success
    "error": "<message>"  // present only on failure
}
```

Example responses:

```json
{"status":"OK", "result": 12}
{"status":"ERROR", "error": "Division by zero"}
```

Fields:

- status: "OK" on success, "ERROR" on error.
- result: computation result.
- error: error description.

The interface prints a simplified text version:

- On success: OK <result_number>
- On error: ERROR <error_message>

## Example Session

```text
$ ./worker
[WORKER] 2026/04/30 16:08:56 Worker process started
[WORKER] 2026/04/30 16:08:59 Worker Listening on ./temp/request.fifo and responding on ./temp/response.fifo
[WORKER] 2026/04/30 16:09:07 Received request: "ADD 3 6\n"
[WORKER] 2026/04/30 16:09:07 Sent response: {"status":"OK","result":9}
[WORKER] 2026/04/30 16:09:15 Received request: "DIV 20 0\n"
[WORKER] 2026/04/30 16:09:15 Sent response: {"status":"ERROR","error":"Division by zero"}
[WORKER] 2026/04/30 16:09:29 Received request: "MUL a b\n"
[WORKER] 2026/04/30 16:09:29 Sent response: {"status":"ERROR","error":"Invalid operand"}
[WORKER] 2026/04/30 16:09:49 Received request: "OP 2 3\n"
[WORKER] 2026/04/30 16:09:49 Sent response: {"status":"ERROR","error":"Invalid operation"}
[WORKER] 2026/04/30 16:09:59 Received request: "ADD 7\n"
[WORKER] 2026/04/30 16:09:59 Sent response: {"status":"ERROR","error":"Invalid arguments"}
[WORKER] 2026/04/30 16:10:47 Received request: "MUL 8 9\n"
[WORKER] 2026/04/30 16:10:47 Sent response: {"status":"OK","result":72}


$ ./interface
[INTERFACE] 2026/04/30 16:08:59 Interface process started
Enter your requests:
> ADD 3 6
OK 9.000000
> DIV 20 0
ERROR Division by zero
> MUL a b
ERROR Invalid operand
> OP 2 3
ERROR Invalid operation
> ADD 7
ERROR Invalid arguments
> MUL 8 9
OK 72.000000
```

## Brief Explanation of Message Passing Protocol

The message passing protocol uses two one way named pipes (FIFOs) to enable full duplex communication between the interface process and the worker process. The interface sends computation requests through the request.fifo pipe as single line text messages in the format OP A B, where OP represents the operation and A and B are numeric operands. Each request is terminated with a newline character so the worker can read it line by line. The worker continuously listens on this FIFO, parses each request, validates the arguments and operation, performs the requested computation and prepares a response.

The response is sent back through a second pipe named response.fifo, using a JSON based message format. Each response contains a status field indicating either "OK" or "ERROR". If the request succeeds, the response includes a result field containing the computed numeric value. if an error occurs such as invalid arguments, division by zero or unsupported operations, the response instead includes an error field with a descriptive message. The interface reads this JSON response, decodes it and prints a output to the user. This bidirectional FIFO setup allows the two independent processes to communicate.