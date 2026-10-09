package main

import (
	"os"
	"fmt"
	"log"
	"math"
	"time"
	"bufio"
	"errors"
	"strconv"
	"strings"
	"syscall"
	"encoding/json"
	"path/filepath"
)

type Response struct {
	Status string  `json:"status"`
	Result float64 `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func checkFifo(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if (info.Mode() & os.ModeNamedPipe) == 0 {
			return fmt.Errorf("%s is not a FIFO", path)
		} else {
			return nil
		}
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	fileDirectoryPath := filepath.Dir(path)
	err = os.MkdirAll(fileDirectoryPath, 0o755)
	if err != nil {
		return err
	}

	err = syscall.Mkfifo(path, 0o666)
	if err != nil {
		return err
	}
	return nil
}

func parseRequest(line string) (string, float64, float64, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return "Empty", 0, 0, errors.New("Invalid arguments")
	}

	operation := strings.ToUpper(fields[0])
	firstOperand, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return "Empty", 0, 0, errors.New("Invalid operand")
	}
	secondOperand, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return "Empty", 0, 0, errors.New("Invalid operand")
	}
	return operation, firstOperand, secondOperand, nil
}

func computeRequest(operation string, firstOperand float64, secondOperand float64) (float64, error) {
	if operation == "ADD" {
		return firstOperand + secondOperand, nil
	} else if operation == "SUB" {
		return firstOperand - secondOperand, nil
	} else if operation == "MUL" {
		return firstOperand * secondOperand, nil
	} else if operation == "DIV" {
		if secondOperand == 0 {
			return 0, errors.New("Division by zero")
		}
		return firstOperand / secondOperand, nil
	} else if operation == "MOD" {
		if secondOperand == 0 {
			return 0, errors.New("Division by zero")
		}
		return math.Mod(firstOperand, secondOperand), nil
	} else if operation == "POW" {
		return math.Pow(firstOperand, secondOperand), nil
	} else if operation == "MAX" {
		if firstOperand > secondOperand {
			return firstOperand, nil
		} else {
			return secondOperand, nil
		}
	} else if operation == "MIN" {
		if firstOperand < secondOperand {
			return firstOperand, nil
		} else {
			return secondOperand, nil
		}
	} else {
		return 0, errors.New("Invalid operation")
	}
}

func handleRequest(line string) Response {
	if line == "" {
		return Response{Status: "ERROR", Error: "Empty request"}
	}

	operation, firstOperand, secondOperand, parseError := parseRequest(line)
	if parseError != nil {
		return Response{Status: "ERROR", Error: parseError.Error()}
	}

	result, computationError := computeRequest(operation, firstOperand, secondOperand)
	if computationError != nil {
		return Response{Status: "ERROR", Error: computationError.Error()}
	}

	return Response{Status: "OK", Result: result}
}


func initLogging() {
	const Green = "\033[32m"
	const Reset = "\033[0m"
	log.SetPrefix(Green + "[WORKER] " + Reset)
	log.Printf("Worker process started")
}

func prepareFifos(requestFifoPath string, responseFifoPath string) {
	requestFifoError := checkFifo(requestFifoPath)
	if requestFifoError != nil {
		log.Fatalf("Failed to create request FIFO: %v", requestFifoError)
	}

	responseFifoError := checkFifo(responseFifoPath)
	if responseFifoError != nil {
		log.Fatalf("Failed to create response FIFO: %v", responseFifoError)
	}
}

func openFifos(requestFifoPath string, responseFifoPath string) (*os.File, *os.File) {
	requestFile, requestFifoError := os.OpenFile(requestFifoPath, os.O_RDONLY, 0)
	if requestFifoError != nil {
		log.Fatalf("Failed to open request FIFO for reading: %v", requestFifoError)
	}

	responsepFile, responseFifoError := os.OpenFile(responseFifoPath, os.O_WRONLY, 0)
	if responseFifoError != nil {
		log.Fatalf("Failed to open response FIFO for writing: %v", responseFifoError)
	}
	return requestFile, responsepFile
}



func processLoop(requestReader *bufio.Reader, resonsepWriter *bufio.Writer, requestFifoPath string, responseFifoPath string) {
	log.Printf("Worker Listening on %s and responding on %s", requestFifoPath, responseFifoPath)

	for {
		line, readError := requestReader.ReadString('\n')
		prepareFifos(requestFifoPath, responseFifoPath)
		
		if readError != nil {
			log.Printf("Request FIFO closed: %v", readError)
			time.Sleep(5*time.Second)
			continue
		}

		log.Printf("Received request: %q", line)
		response := handleRequest(line)
		data, _ := json.Marshal(response)

		_, writeError := resonsepWriter.Write(append(data, '\n'))
		if writeError != nil {
			log.Printf("Failed to write response: %v", writeError)
			continue
		}

		flushError := resonsepWriter.Flush()
		if flushError != nil {
			log.Printf("Failed to flush response: %v", flushError)
			continue
		}

		log.Printf("Sent response: %s", string(data))
	}
}

func main() {
	requestFifoPath := "./temp/request.fifo"
	responseFifoPath := "./temp/response.fifo"

	initLogging()
	prepareFifos(requestFifoPath, responseFifoPath)

	requestFile, responseFile := openFifos(requestFifoPath, responseFifoPath)
	defer requestFile.Close()
	defer responseFile.Close()

	requestReader := bufio.NewReader(requestFile)
	responseWriter := bufio.NewWriter(responseFile)

	processLoop(requestReader, responseWriter, requestFifoPath, responseFifoPath)
}
