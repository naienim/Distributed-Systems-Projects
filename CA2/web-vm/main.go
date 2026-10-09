package main

import (
	"fmt"
	"log"
	"net/http"
	"html/template"
	"encoding/base64"
	"net/rpc/jsonrpc"
)

const (
	authHost = "192.168.137.131"
	authPort = "8081"

	fileHost = "192.168.137.132"
	filePort = "9090"
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
	Username    string
	ImageData   template.URL
	TextContent string
}

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
		http.Error(w, "Could not connect to authentication server: " + err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "RPC call failed: " + err.Error(), http.StatusInternalServerError)
		return
	}

	if !result.Success {
		loginTemplate.Execute(w, map[string]string{"Error": result.Message})
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

func main() {
	http.HandleFunc("/", showLoginPage)
	http.HandleFunc("/login", handleLogin)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Web server running on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
