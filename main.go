package main

import (
	"database/sql"
	"encoding/json"
	"httpserv/internal/database"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	PLATFORM       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited " + strconv.Itoa(int(hits)) + " times!</p></body></html>"))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	if cfg.PLATFORM != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	err := cfg.dbQueries.DeleteUsers(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete user", "details": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "all users deleted.", "reset": "Hits reset to 0"})

}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) apiuser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type Email struct {
		Emailid string `json:"email"`
	}
	var email Email
	if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	// email stored

	user, err := cfg.dbQueries.CreateUser(r.Context(), email.Emailid)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"email":      user.Email,
	})

}

type Chirp struct {
	Body    string    `json:"body"`
	User_id uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) post(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var chirp Chirp
	if err := json.NewDecoder(r.Body).Decode(&chirp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if len(chirp.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Chirp is too long"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	post, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   chirp.Body,
		UserID: chirp.User_id,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "error making db request,", "details": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         post.ID,
		"created_at": post.CreatedAt,
		"updated_at": post.UpdatedAt,
		"body":       post.Body,
		"user_id":    post.UserID,
	})

}

func (cfg *apiConfig) getchirps(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	posts, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "error fetching,", "details": err.Error()})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(posts)
}

func (cfg *apiConfig) specchirps(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Extracts `{id}` from the pat
	uuid, err := uuid.Parse(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid UUID format", "details": err.Error()})
		return
	}

	post, err := cfg.dbQueries.GetPost(r.Context(), uuid)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "error getting id,", "details": err.Error()})
	}
	json.NewEncoder(w).Encode(post)
}

func HttpServer() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		dbQueries: dbQueries,
		PLATFORM:  os.Getenv("PLATFORM"),
	}
	mux := http.NewServeMux()
	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	mux.HandleFunc("GET /api/healthz", readinessHandler)                                                       // check status
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("."))))) // deliver files
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)                                                // metics
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)                                                   // reset usewrs
	mux.HandleFunc("POST /api/users", apiCfg.apiuser)                                                          // add user
	mux.HandleFunc("POST /api/chirps", apiCfg.post)                                                            // add post
	mux.HandleFunc("GET /api/chirps", apiCfg.getchirps)                                                        // get post of user
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.specchirps)

	log.Println("Starting server on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func main() {
	HttpServer()
}
