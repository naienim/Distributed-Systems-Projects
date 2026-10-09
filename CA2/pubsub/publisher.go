package main

import (
	"fmt"
	"log"
	"net"
	"time"
	"runtime"
	"net/http"
	"encoding/json"
	"html/template"
	"net/rpc/jsonrpc"
	"encoding/base64"
)

const (
	authHost = "192.168.137.131"
	authPort = "8081"

	fileHost = "192.168.137.132"
	filePort = "9090"

	subHost = "192.168.137.130"
	subPort = "7000"

	thresholdMB = 300
	monitorIntervalSecond = 3
)

type LoginRequest struct {
	Username string
	Password string
}

type LoginResponse struct {
	Success bool
	Message string
}

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

type WelcomePageData struct {
	Username string
	ImageData template.URL
	TextContent string
}

var allocatedBlocks [][]byte
var loginTemplate = template.Must(template.ParseFiles("templates/login.html"))
var welcomeTemplate = template.Must(template.ParseFiles("templates/welcome.html"))

func showLoginPage(w http.ResponseWriter, r *http.Request) {
	loginTemplate.Execute(w, nil)
}

func getFileFromRPC(path string) (FileResponse, error) {
	fileServerAddress := fileHost + ":" + filePort

	client, err := jsonrpc.Dial("tcp", fileServerAddress)
	if err != nil {
		return FileResponse{}, err
	}
	defer client.Close()

	request := FileRequest{Path: path}
	var response FileResponse

	err = client.Call("FileService.GetFile", request, &response)
	if err != nil {
		return FileResponse{}, err
	}

	return response, nil
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	authenticationServerAddress := authHost + ":" + authPort

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	client, err := jsonrpc.Dial("tcp", authenticationServerAddress)
	if err != nil {
		http.Error(w, "Could not connect to authentication server: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	request := LoginRequest{
		Username: username,
		Password: password,
	}

	var result LoginResponse

	err = client.Call("AuthenticationService.Login", request, &result)
	if err != nil {
		http.Error(w, "RPC call failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !result.Success {
		loginTemplate.Execute(w, map[string]string{
			"Error": result.Message,
		})
		return
	}

	imageResponse, imageErr := getFileFromRPC("images/sample.png")
	if imageErr != nil || !imageResponse.Success {
		http.Error(w, "Could not get image from file RPC server", http.StatusInternalServerError)
		return
	}

	textResponse, textErr := getFileFromRPC("files/sample.txt")
	if textErr != nil || !textResponse.Success {
		http.Error(w, "Could not get text file from file RPC server", http.StatusInternalServerError)
		return
	}

	decodedTextBytes, decodeErr := base64.StdEncoding.DecodeString(textResponse.Data)
	if decodeErr != nil {
		http.Error(w, "Could not decode text file", http.StatusInternalServerError)
		return
	}

	imageDataURI := "data:" + imageResponse.ContentType + ";base64," + imageResponse.Data

	welcomeTemplate.Execute(w, WelcomePageData{
		Username:    username,
		ImageData:   template.URL(imageDataURI),
		TextContent: string(decodedTextBytes),
	})
}

func consumeMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var sizeMB int

	mbParameter := r.URL.Query().Get("mb")
	if mbParameter == "" {
		http.Error(w, "missing required query parameter: mb", http.StatusBadRequest)
		return
	}

	_, parseError := fmt.Sscanf(mbParameter, "%d", &sizeMB)
	if parseError != nil || sizeMB <= 0 {
		http.Error(w, "invalid 'mb' query parameter", http.StatusBadRequest)
		return
	}

	bytesToAllocate := sizeMB * 1024 * 1024
	block := make([]byte, bytesToAllocate)

	const pageSize = 4096
	for offset := 0; offset < len(block); offset += pageSize {
		block[offset] = 1
	}

	allocatedBlocks = append(allocatedBlocks, block)

	fmt.Fprintf(w, "Allocated %d MB. Total allocations=%d\n", sizeMB, len(allocatedBlocks))
}

func startMemoryMonitor() {
	alreadyAlerted := false

	for {
		var memoryStats runtime.MemStats
		runtime.ReadMemStats(&memoryStats)

		memoryMB := memoryStats.HeapAlloc / (1024 * 1024)

		if memoryMB > thresholdMB && !alreadyAlerted {
			event := map[string]any{
				"event_type":   "HIGH_MEMORY_USAGE",
				"service":      "web-server",
				"memory_mb":    memoryMB,
				"threshold_mb": thresholdMB,
				"timestamp":    time.Now(),
			}

			publishError := publishEvent(event)
			if publishError != nil {
				log.Println("Publish failed:", publishError)
			} else {
				log.Println("High memory usage event published")
				alreadyAlerted = true
			}
		} else if memoryMB <= thresholdMB {
			alreadyAlerted = false
		}

		time.Sleep(time.Second * monitorIntervalSecond)
	}
}

func publishEvent(event map[string]any) error {
	subscriberAddress := subHost + ":" + subPort

	messageBytes, marshalError := json.Marshal(event)
	if marshalError != nil {
		return marshalError
	}

	connection, connectionError := net.Dial("tcp", subscriberAddress)
	if connectionError != nil {
		return connectionError
	}
	defer connection.Close()

	_, writeError := connection.Write(append(messageBytes, '\n'))
	return writeError
}

func main() {
	go startMemoryMonitor()

	http.HandleFunc("/", showLoginPage)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/consume-memory", consumeMemoryHandler)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Web server running on port 8080...")
	fmt.Println("Memory monitor is running...")
	fmt.Println("Use /consume-memory?mb=100 to increase memory usage.")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
