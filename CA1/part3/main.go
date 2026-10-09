package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

type ComputeResponse struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
	Result    float64 `json:"result"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
		Error: "Method Not Allowed!",
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	response := HealthResponse{
		Status:  "ok",
		Service: "Distributed Systems HW1 Part3",
		Time:    time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, response)
}

func parseParam(r *http.Request, name string) (float64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, fmt.Errorf("missing parameter: %s", name)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parameter %s must be numeric", name)
	}
	return number, nil
}

func compute(operator string, a float64, b float64) (float64, error) {
	switch strings.ToLower(operator) {
	case "add":
		return a + b, nil
	case "sub":
		return a - b, nil
	case "mul":
		return a * b, nil
	case "div":
		if b == 0 {
			return 0, errors.New("Division By Zero!")
		}
		return a / b, nil
	case "mod":
		if b == 0 {
			return 0, errors.New("Division By Zero!")
		}
		return math.Mod(a, b), nil
	case "pow":
		return math.Pow(a, b), nil

	default:
		return 0, errors.New("Invalid Operation!")
	}
}

func computeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query := r.URL.Query()
	operator := query.Get("operator")
	if operator == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "missing parameter: operator",
		})
		return
	}
	a, err := parseParam(r, "a")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	b, err := parseParam(r, "b")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := compute(operator, a, b)
	if err != nil {
		if err.Error() == "Division By Zero!" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "Division By Zero!",
			})
			return
		}

		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "Invalid Operation!",
		})
		return
	}

	response := ComputeResponse{
		Operation: strings.ToLower(operator),
		A:         a,
		B:         b,
		Result:    result,
	}

	writeJSON(w, http.StatusOK, response)
}

func logOutput(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)
		next(w, r)
	}
}

func main() {
	route := http.NewServeMux()
	route.HandleFunc("/health", logOutput(healthHandler))
	route.HandleFunc("/compute", logOutput(computeHandler))
	port := "8080"
	address := ":" + port
	log.Printf("HTTP service is running on port %s", port)
	err := http.ListenAndServe(address, route)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}