package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Config struct {
	ID                 string   `json:"id"`
	Port               int      `json:"port"`
	Peers              []string `json:"peers"`
	ReplicationDelayMS int      `json:"replication_delay_ms"`
	DataFile           string   `json:"data_file"`
}

type Record struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Version   int    `json:"version"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt int64  `json:"updated_at"`
}

type PutRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Mode  string `json:"mode"` 
}

type PutResponse struct {
	OK              bool    `json:"ok"`
	Message         string  `json:"message"`
	Record          Record  `json:"record"`
	UpdatedReplicas int     `json:"updated_replicas"`
	LatencyMS       float64 `json:"latency_ms"`
}

type GetResponse struct {
	OK              bool    `json:"ok"`
	Message         string  `json:"message"`
	Record          Record  `json:"record"`
	CheckedReplicas int     `json:"checked_replicas"`
	LatencyMS       float64 `json:"latency_ms"`
}

type InternalResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Record  Record `json:"record"`
}

type Server struct {
	cfg  Config
	mu   sync.Mutex
	data map[string]Record
}

func main() {
	configPath := flag.String("config", "", "path to replica config json")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("usage: go run replica/main.go -config configs/replica1.json")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	if cfg.DataFile == "" {
		cfg.DataFile = fmt.Sprintf("%s_data.json", cfg.ID)
	}

	s := &Server{cfg: cfg, data: map[string]Record{}}
	if err := s.loadData(); err != nil {
		log.Printf("starting with empty data store: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/put", s.handlePut)
	mux.HandleFunc("/get", s.handleGet)
	mux.HandleFunc("/dump", s.handleDump)
	mux.HandleFunc("/internal/replicate", s.handleInternalReplicate)
	mux.HandleFunc("/internal/get", s.handleInternalGet)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("%s is running on %s", cfg.ID, addr)
	log.Printf("peers: %v", cfg.Peers)
	log.Printf("replication delay: %d ms", cfg.ReplicationDelayMS)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	return cfg, json.Unmarshal(b, &cfg)
}

func (s *Server) loadData() error {
	b, err := os.ReadFile(s.cfg.DataFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *Server) saveDataLocked() {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		log.Printf("cannot marshal data: %v", err)
		return
	}
	if err := os.WriteFile(s.cfg.DataFile, b, 0644); err != nil {
		log.Printf("cannot save data: %v", err)
	}
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	start := time.Now()

	var req PutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	if req.Mode == "" {
		req.Mode = "eventual"
	}

	s.mu.Lock()
	old := s.data[req.Key]
	nextVersion := old.Version + 1
	rec := Record{
		Key:       req.Key,
		Value:     req.Value,
		Version:   nextVersion,
		UpdatedBy: s.cfg.ID,
		UpdatedAt: time.Now().UnixNano(),
	}
	s.mu.Unlock()

	switch req.Mode {
	case "eventual":
		s.mu.Lock()
		s.data[rec.Key] = rec
		s.saveDataLocked()
		s.mu.Unlock()
		go s.replicateToPeers(rec)
		writeJSON(w, http.StatusOK, PutResponse{
			OK:              true,
			Message:         "PUT applied locally; replication is running asynchronously",
			Record:          rec,
			UpdatedReplicas: 1,
			LatencyMS:       float64(time.Since(start).Microseconds()) / 1000.0,
		})

	case "strong":
		peerAcks := s.replicateToPeers(rec)
		updatedReplicas := 1 + peerAcks

		if updatedReplicas >= 2 {
			s.mu.Lock()
			s.data[rec.Key] = rec
			s.saveDataLocked()
			s.mu.Unlock()

			writeJSON(w, http.StatusOK, PutResponse{
				OK:              true,
				Message:         "PUT committed with majority acknowledgement",
				Record:          rec,
				UpdatedReplicas: updatedReplicas,
				LatencyMS:       float64(time.Since(start).Microseconds()) / 1000.0,
			})
			return
		}

		writeJSON(w, http.StatusServiceUnavailable, PutResponse{
			OK:              false,
			Message:         "PUT failed: majority of replicas is not available",
			Record:          rec,
			UpdatedReplicas: updatedReplicas,
			LatencyMS:       float64(time.Since(start).Microseconds()) / 1000.0,
		})

	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mode must be eventual or strong"})
	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	key := r.URL.Query().Get("key")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "eventual"
	}
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}

	s.mu.Lock()
	local, ok := s.data[key]
	s.mu.Unlock()

	if mode == "eventual" {
		if !ok {
			writeJSON(w, http.StatusNotFound, GetResponse{OK: false, Message: "key not found", LatencyMS: float64(time.Since(start).Microseconds()) / 1000.0})
			return
		}
		writeJSON(w, http.StatusOK, GetResponse{
			OK:              true,
			Message:         "local read from this replica",
			Record:          local,
			CheckedReplicas: 1,
			LatencyMS:       float64(time.Since(start).Microseconds()) / 1000.0,
		})
		return
	}

	if mode != "strong" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mode must be eventual or strong"})
		return
	}

	latest := local
	found := ok
	checked := 1

	for _, peer := range s.cfg.Peers {
		if rec, peerOK := s.fetchPeerRecord(peer, key); peerOK {
			checked++
			if !found || isBetter(rec, latest) {
				latest = rec
				found = true
			}
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, GetResponse{OK: false, Message: "key not found", CheckedReplicas: checked, LatencyMS: float64(time.Since(start).Microseconds()) / 1000.0})
		return
	}

	s.mu.Lock()
	changed, _ := s.applyRecordLocked(latest)
	if changed {
		s.saveDataLocked()
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, GetResponse{
		OK:              true,
		Message:         "coordinated strong read; newest version returned",
		Record:          latest,
		CheckedReplicas: checked,
		LatencyMS:       float64(time.Since(start).Microseconds()) / 1000.0,
	})
}

func (s *Server) handleInternalReplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	var rec Record
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	changed, msg := s.applyRecordLocked(rec)
	current := s.data[rec.Key]
	if changed {
		s.saveDataLocked()
	}
	s.mu.Unlock()
	ack := recordsEqual(current, rec)
	writeJSON(w, http.StatusOK, InternalResponse{OK: ack, Message: msg, Record: current})
}

func (s *Server) handleInternalGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	s.mu.Lock()
	rec, ok := s.data[key]
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, InternalResponse{OK: false, Message: "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, InternalResponse{OK: true, Message: "ok", Record: rec})
}

func (s *Server) handleDump(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.data)
}

func (s *Server) replicateToPeers(rec Record) int {
	var wg sync.WaitGroup
	ackCh := make(chan bool, len(s.cfg.Peers))

	for _, peer := range s.cfg.Peers {
		peer := peer
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.cfg.ReplicationDelayMS > 0 {
				time.Sleep(time.Duration(s.cfg.ReplicationDelayMS) * time.Millisecond)
			}
			ackCh <- s.sendReplication(peer, rec)
		}()
	}

	wg.Wait()
	close(ackCh)

	acks := 0
	for ok := range ackCh {
		if ok {
			acks++
		}
	}
	return acks
}

func (s *Server) sendReplication(peer string, rec Record) bool {
	b, _ := json.Marshal(rec)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(peer+"/internal/replicate", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Printf("replication to %s failed: %v", peer, err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var out InternalResponse
	if err := json.Unmarshal(body, &out); err != nil {
		log.Printf("bad response from %s: %s", peer, string(body))
		return false
	}
	return resp.StatusCode == http.StatusOK && out.OK
}

func (s *Server) fetchPeerRecord(peer, key string) (Record, bool) {
	client := &http.Client{Timeout: 2 * time.Second}
	endpoint := peer + "/internal/get?key=" + url.QueryEscape(key)
	resp, err := client.Get(endpoint)
	if err != nil {
		return Record{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Record{}, false
	}
	var out InternalResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Record{}, false
	}
	return out.Record, out.OK
}

func (s *Server) applyRecordLocked(in Record) (bool, string) {
	old, exists := s.data[in.Key]
	if !exists {
		s.data[in.Key] = in
		return true, "new record stored"
	}
	if in.Version > old.Version {
		s.data[in.Key] = in
		return true, "newer version stored"
	}
	if in.Version < old.Version {
		return false, "ignored older version"
	}

	if recordsEqual(old, in) {
		return false, "duplicate record ignored"
	}


	if isBetter(in, old) {
		s.data[in.Key] = in
		return true, "conflict resolved: larger replica id won"
	}
	return false, "conflict resolved: existing record won"
}

func recordsEqual(a, b Record) bool {
	return a.Key == b.Key && a.Value == b.Value && a.Version == b.Version && a.UpdatedBy == b.UpdatedBy
}

func isBetter(a, b Record) bool {
	if a.Version != b.Version {
		return a.Version > b.Version
	}
	if a.UpdatedBy != b.UpdatedBy {
		return a.UpdatedBy > b.UpdatedBy
	}
	return a.UpdatedAt > b.UpdatedAt
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
