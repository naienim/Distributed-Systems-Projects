package main

import (
	"os"
	"fmt"
	"log"
	"time"
	"bufio"
	"errors"
	"strings"
	"syscall"
	"encoding/json"
)

type Response struct {
	Status string  `json:"status"`
	Result float64 `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		_, err := os.Stat(path)
		if err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if time.Now().After(deadline) {
			return errors.New("timeout waiting for " + path)
		}
		time.Sleep(200*time.Millisecond)
	}
}

func initLogging() {
	const Yellow = "\033[33m"
	const Reset = "\033[0m"
	log.SetPrefix(Yellow + "[INTERFACE] " + Reset)
	log.Printf("Interface process started")
}

func waitForFifos(requestFifoPath string, responseFifoPath string) {
	for {
		requestFifoError := waitForFile(requestFifoPath, 5*time.Second)
		if requestFifoError != nil {
			// log.Fatalf("Request FIFO not found: %v", requestFifoError)
			log.Printf("Request FIFO not found: %v", requestFifoError)
			continue
		}
	
		responseFifoError := waitForFile(responseFifoPath, 5*time.Second)
		if responseFifoError != nil {
			// log.Fatalf("Response FIFO not found: %v", responseFifoError)
			log.Printf("Response FIFO not found: %v", responseFifoError)
			continue
		}

		return
	}
}

func openFifos(requestFifoPath string, responseFifoPath string) (*os.File, *os.File) {
	requestFile, requestFifoError := os.OpenFile(requestFifoPath, os.O_WRONLY, 0)
	if requestFifoError != nil {
		log.Fatalf("Failed to open request FIFO for writing: %v", requestFifoError)
	}

	responseFile, responseFifoError := os.OpenFile(responseFifoPath, os.O_RDONLY, 0)
	if responseFifoError != nil {
		log.Fatalf("Failed to open response FIFO for reading: %v", responseFifoError)
	}

	return requestFile, responseFile
}


func blockingReOpenFifos(requestFifoPath string, responseFifoPath string) (*bufio.Writer, *bufio.Reader, error) {
    requestFile, requestFifoError := os.OpenFile(requestFifoPath, os.O_WRONLY, 0)
    if requestFifoError != nil {
        return nil, nil, requestFifoError
    }

    responseFile, responseFifoError := os.OpenFile(responseFifoPath, os.O_RDONLY, 0)
    if responseFifoError != nil {
        requestFile.Close()
        return nil, nil, responseFifoError
    }

    requestWriter := bufio.NewWriter(requestFile)
    responseReader := bufio.NewReader(responseFile)
    return requestWriter, responseReader, nil
}



func nonBlockingReOpenFifos(requestFifoPath string, responseFifoPath string) (*bufio.Writer, *bufio.Reader, error) {

	requestFd, requestFifoError := syscall.Open(requestFifoPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if requestFifoError != nil {
		return nil, nil, requestFifoError
	}

	responseFd, responseFifoError := syscall.Open(responseFifoPath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if responseFifoError != nil {
		syscall.Close(requestFd)
		return nil, nil, responseFifoError
	}

	requestFile := os.NewFile(uintptr(requestFd), requestFifoPath)
	responseFile := os.NewFile(uintptr(responseFd), responseFifoPath)

	requestWriter := bufio.NewWriter(requestFile)
	responseReader := bufio.NewReader(responseFile)

	return requestWriter, responseReader, nil
}



func processUserLoop(requestWriter *bufio.Writer, responseReader *bufio.Reader, stdinReader *bufio.Reader, requestFifoPath string, responseFifoPath string) {
	fmt.Printf("Enter your requests:\n")
	fifoIsClose := false
	
	for {
		fmt.Print("> ")
		var response Response
		line, _ := stdinReader.ReadString('\n')

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if fifoIsClose == true {
			requestWriterTemp, responseReaderTemp, reOPenError := nonBlockingReOpenFifos(requestFifoPath, responseFifoPath)
			if reOPenError == nil {
				requestWriter, responseReader = requestWriterTemp, responseReaderTemp
				fifoIsClose = false
			} else {
				log.Printf("Failed to open FIFOs: %v", reOPenError)
				fifoIsClose = true
				continue
			}
		}

		_, writeError := requestWriter.WriteString(line + "\n")
		if writeError != nil {
			log.Printf("Failed to write request: %v", writeError)
			fifoIsClose = true
			continue
		}

		flushError := requestWriter.Flush()
		if flushError != nil {
			log.Printf("Failed to flush request: %v", flushError)
			fifoIsClose = true
			continue
		}

		responseLine, readError := responseReader.ReadString('\n')
		if readError != nil {
			log.Printf("Failed to read response: %v", readError)
			fifoIsClose = true
			continue
		}


		json.Unmarshal([]byte(responseLine), &response)
		if response.Status == "OK" {
			fmt.Printf("OK %.2f\n", response.Result)
		} else {
			fmt.Printf("ERROR %s\n", response.Error)
		}
	}
}

func main() {
	requestFifoPath := "./temp/request.fifo"
	responseFifoPath := "./temp/response.fifo"

	initLogging()
	waitForFifos(requestFifoPath, responseFifoPath)

	requestFile, responseFile := openFifos(requestFifoPath, responseFifoPath)
	defer requestFile.Close()
	defer responseFile.Close()

	requesstWriter := bufio.NewWriter(requestFile)
	responsepReader := bufio.NewReader(responseFile)
	stdinReader := bufio.NewReader(os.Stdin)
	processUserLoop(requesstWriter, responsepReader, stdinReader, requestFifoPath, responseFifoPath)
}
