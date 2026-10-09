package main

import (
	"os"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"path/filepath"
	"net/rpc/jsonrpc"
	"encoding/base64"
)

type FileRequest struct {
	Path string
}

type FileResponse struct {
	Success bool
	Message string
	FileName string
	ContentType string
	Data string
}

type FileService struct {
	BaseDir string
}

func (f *FileService) GetFile(request FileRequest, response *FileResponse) error {
	cleanPath := filepath.Clean(request.Path)
	fullPath := filepath.Join(f.BaseDir, cleanPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		response.Success = false
		response.Message = "Could not read file: " + err.Error()
		return nil
	}

	response.Success = true
	response.Message = "File loaded successfully"
	response.FileName = filepath.Base(fullPath)
	response.Data = base64.StdEncoding.EncodeToString(data)

	extension := filepath.Ext(fullPath)
	switch extension {
		case ".png":
			response.ContentType = "image/png"
		case ".jpg", ".jpeg":
			response.ContentType = "image/jpeg"
		case ".gif":
			response.ContentType = "image/gif"
		case ".txt":
			response.ContentType = "text/plain"
		default:
			response.ContentType = "application/octet-stream"
	}

	log.Println("File sent through RPC:", request.Path)
	return nil
}

func main() {
	fileService := &FileService{BaseDir: "."}

	err := rpc.RegisterName("FileService", fileService)
	if err != nil {
		log.Fatal("Could not register FileService:", err)
	}

	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal("Could not start file RPC server:", err)
	}
	defer listener.Close()

	fmt.Println("File RPC server running on port 9090...")

	for {
		connection, connectionError := listener.Accept()
		if connectionError != nil {
			log.Println("Connection error:", connectionError)
			continue
		}

		go rpc.ServeCodec(jsonrpc.NewServerCodec(connection))
	}
}