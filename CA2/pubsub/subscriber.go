package main

import (
	"fmt"
	"log"
	"net"
	"bufio"
	"encoding/json"
)

type Event struct {
	EventType string `json:"event_type"`
	Service string `json:"service"`
	MemoryMB uint64 `json:"memory_mb"`
	ThresholdMB uint64 `json:"threshold_mb"`
	Timestamp string `json:"timestamp"`
}

func handleConnection(connection net.Conn) {
	defer connection.Close()

	reader := bufio.NewReader(connection)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		log.Printf("read error from %s: %v", connection.RemoteAddr(), err)
		return
	}

	var event Event
	json.Unmarshal(line, &event);
	printAlert(event)
}

func printAlert(event Event) {
	fmt.Println("===================================================")
	fmt.Println("ALERT: HIGH MEMORY USAGE RECEIVED")
	fmt.Println("Service       :", event.Service)
	fmt.Println("Memory (MB)   :", event.MemoryMB)
	fmt.Println("Threshold (MB):", event.ThresholdMB)
	fmt.Println("Timestamp     :", event.Timestamp)
	fmt.Println("===================================================")
}

func main() {
	listener, _ := net.Listen("tcp", ":7000")
	defer listener.Close()

	fmt.Println("Subscriber listening on port 7000...")
	fmt.Println("Waiting for high memory usage events...")

	for {
		connection, connectionError := listener.Accept()
		if connectionError != nil {
			log.Println("accept error:", connectionError)
			continue
		}
		go handleConnection(connection)
	}
}
