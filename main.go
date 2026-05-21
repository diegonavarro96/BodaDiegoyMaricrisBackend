package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RSVP struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Attending  string    `json:"attending"`
	Vegetarian bool      `json:"vegetarian"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
			return nil, err
		}
	}
	return &Store{path: path}, nil
}

func (s *Store) load() ([]RSVP, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []RSVP{}, nil
	}
	var rsvps []RSVP
	if err := json.Unmarshal(data, &rsvps); err != nil {
		return nil, err
	}
	return rsvps, nil
}

func (s *Store) save(rsvps []RSVP) error {
	data, err := json.MarshalIndent(rsvps, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) List() ([]RSVP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// UpsertByEmail inserts r, or if a record with the same email (case-
// insensitive) already exists, replaces it in place — keeping the original
// ID and CreatedAt so admin views stay stable. Returns the persisted record
// and whether it was an update of an existing row.
func (s *Store) UpsertByEmail(r RSVP) (RSVP, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rsvps, err := s.load()
	if err != nil {
		return RSVP{}, false, err
	}
	needle := strings.ToLower(r.Email)
	for i, existing := range rsvps {
		if strings.ToLower(existing.Email) == needle {
			r.ID = existing.ID
			r.CreatedAt = existing.CreatedAt
			rsvps[i] = r
			return r, true, s.save(rsvps)
		}
	}
	rsvps = append(rsvps, r)
	return r, false, s.save(rsvps)
}

func (s *Store) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rsvps, err := s.load()
	if err != nil {
		return false, err
	}
	for i, r := range rsvps {
		if r.ID == id {
			rsvps = append(rsvps[:i], rsvps[i+1:]...)
			return true, s.save(rsvps)
		}
	}
	return false, nil
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

type Server struct {
	store         *Store
	adminToken    string
	allowedOrigin string
}

func (s *Server) cors(w http.ResponseWriter, r *http.Request) bool {
	origin := s.allowedOrigin
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func (s *Server) requireAdmin(r *http.Request) bool {
	if s.adminToken == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := strings.TrimPrefix(h, prefix)
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.adminToken)) == 1
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func validEmail(s string) bool {
	if len(s) < 3 || len(s) > 254 {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at < 1 || at == len(s)-1 {
		return false
	}
	if strings.IndexByte(s[at+1:], '.') < 0 {
		return false
	}
	return true
}

func (s *Server) handleRSVP(w http.ResponseWriter, r *http.Request) {
	if s.cors(w, r) {
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.createRSVP(w, r)
	case http.MethodGet:
		if !s.requireAdmin(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		rsvps, err := s.store.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read rsvps")
			return
		}
		writeJSON(w, http.StatusOK, rsvps)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) createRSVP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Attending  string `json:"attending"`
		Vegetarian bool   `json:"vegetarian"`
		Message    string `json:"message"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	in.Message = strings.TrimSpace(in.Message)

	if in.Name == "" || len(in.Name) > 200 {
		writeError(w, http.StatusBadRequest, "name required (1-200 chars)")
		return
	}
	if !validEmail(in.Email) {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	if in.Attending != "yes" && in.Attending != "no" {
		writeError(w, http.StatusBadRequest, "attending must be 'yes' or 'no'")
		return
	}
	if len(in.Message) > 2000 {
		writeError(w, http.StatusBadRequest, "message too long")
		return
	}

	rsvp := RSVP{
		ID:         newID(),
		Name:       in.Name,
		Email:      in.Email,
		Attending:  in.Attending,
		Vegetarian: in.Vegetarian,
		Message:    in.Message,
		CreatedAt:  time.Now().UTC(),
	}
	saved, updated, err := s.store.UpsertByEmail(rsvp)
	if err != nil {
		log.Printf("save rsvp: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save rsvp")
		return
	}
	status := http.StatusCreated
	if updated {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"id": saved.ID, "updated": updated})
}

func (s *Server) handleRSVPByID(w http.ResponseWriter, r *http.Request) {
	if s.cors(w, r) {
		return
	}
	if !s.requireAdmin(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/rsvp/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ok, err := s.store.Delete(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := envOr("PORT", "8080")
	dataPath := envOr("DATA_PATH", "data/rsvps.json")
	adminToken := os.Getenv("ADMIN_TOKEN")
	allowedOrigin := envOr("ALLOWED_ORIGIN", "http://localhost:5173")

	if adminToken == "" {
		log.Println("WARNING: ADMIN_TOKEN is not set — admin endpoints will reject all requests")
	}

	store, err := NewStore(dataPath)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	srv := &Server{store: store, adminToken: adminToken, allowedOrigin: allowedOrigin}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/rsvp", srv.handleRSVP)
	mux.HandleFunc("/api/rsvp/", srv.handleRSVPByID)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	addr := ":" + port
	log.Printf("RSVP server listening on %s (origin allowed: %s, data: %s)", addr, allowedOrigin, dataPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
