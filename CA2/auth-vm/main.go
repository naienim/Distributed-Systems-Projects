package main

import (
	"os"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/rpc/jsonrpc"
)

type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type LoginRequest struct {
	Username string
	Password string
}

type LoginResponse struct {
	Success bool
	Message string
}

type AuthenticationService struct {
	Users []User
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func (a *AuthenticationService) Login(request LoginRequest, result *LoginResponse) error {
	requestPasswordHash := hashPassword(request.Password)
	for _, user := range a.Users {
		if user.Username == request.Username && user.PasswordHash == requestPasswordHash {
			result.Success = true
			result.Message = "Login successful"
			log.Println("Successful login for user:", request.Username)
			return nil
		}
	}

	result.Success = false
	result.Message = "Invalid username or password"
	log.Println("Failed login for user:", request.Username)
	return nil
}

func loadUsers(filename string) ([]User, error) {
	var users []User
	data, fileError := os.ReadFile(filename)
	if fileError != nil {
		log.Printf("Could not read users file: %v", fileError)
		return users, fileError
	}
	json.Unmarshal(data, &users)
	return users, nil
}

func main() {
	users, _ := loadUsers("users.json")

	authService := &AuthenticationService{Users: users}
	rpc.RegisterName("AuthenticationService", authService);
	listener, _ := net.Listen("tcp", ":8081")
	defer listener.Close()

	fmt.Println("Authentication server running on port 8081...")

	for {
		connection, connectionError := listener.Accept()
		if connectionError != nil {
			log.Println("Connection error:", connectionError)
			continue
		}
		go rpc.ServeCodec(jsonrpc.NewServerCodec(connection))
	}
}