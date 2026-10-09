# File VM (VM3) — JSON-RPC File and Image Service

This service stores files and images locally on VM3 and provides them to VM1 through JSON-RPC. VM1 calls this service after a successful login and receives the requested file content through the network.


## Responsibilities

- Host sample files and images on VM3
- Expose a JSON-RPC method for retrieving files
- Read requested files from VM3 local storage
- Return file content to VM1 as base64 encoded data
- Allow VM1 to display file and image content without accessing VM3 files directly from disk

## Files and folders

- main.go: JSON-RPC file server
- files/: text files and other documents such as sample.txt
- images/: image files such as sample.png

## Requirements

- Go installed on Linux
- Port 9090 open on VM3
- files/ and images/ directories must exist
- VM1 must point to the correct VM3 IP address

## Run

From inside the file-vm/ directory:

```bash
go run main.go
```

The RPC server listens on port 9090.

## RPC method

The exposed RPC method is:

```text
FileService.GetFile
```

Request fields:

- path

Response fields:

- success
- message
- fileName
- contentType
- data

The data field contains the file content encoded as base64.

## File paths

VM1 requests files using relative paths such as:

```text
files/sample.txt
images/sample.png
```

VM3 then reads these files from its own local folders:

```text
./files/sample.txt
./images/sample.png
```

## How it works

The server runs a JSON-RPC service on port 9090. When VM1 requests a file, VM3 reads the file from its local disk, detects the file content type based on its extension, encodes the file data using base64, and sends the result back to VM1 through RPC.

VM1 does not access these files directly from its own disk. It also does not load them using public HTTP URLs anymore. The file content is transferred through JSON-RPC.

## Testing

The normal way to test this service is through VM1’s web page.

After a successful login, VM1 should call VM3 through JSON-RPC and display:

- the text content from files/sample.txt
- the image content from images/sample.png

You can also check the VM3 terminal logs. When a file is successfully sent, the server prints a log message showing the requested path.
