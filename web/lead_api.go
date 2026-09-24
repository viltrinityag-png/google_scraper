package web

import (
	"crypto/subtle"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type leadSearchRequest struct {
	Business       string `json:"business"`
	Location       string `json:"location"`
	Limit          int    `json:"limit,omitempty"`
	Email          bool   `json:"email,omitempty"`
	FastMode       bool   `json:"fast_mode,omitempty"`
	Depth          int    `json:"depth,omitempty"`
	Radius         int    `json:"radius,omitempty"`
	Lang           string `json:"lang,omitempty"`
	Zoom           int    `json:"zoom,omitempty"`
	GeoCoordinates string `json:"geo_coordinates,omitempty"`
	MaxTime        int    `json:"max_time,omitempty"`
}

type leadSearchCreated struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Search    string `json:"search"`
	ResultURL string `json:"result_url"`
}

type leadSearchResult struct {
	ID        string              `json:"id"`
	Status    string              `json:"status"`
	Search    string              `json:"search"`
	Count     int                 `json:"count"`
	Requested int                 `json:"requested"`
	Results   []map[string]string `json:"results,omitempty"`
	Error     string              `json:"error,omitempty"`
}

func leadAPIKeyMiddleware(next http.Handler) http.Handler {
	configured := strings.TrimSpace(os.Getenv("LEAD_API_KEY"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if configured == "" {
			next.ServeHTTP(w, r)
			return
		}
		supplied := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if supplied == "" {
			supplied = strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		}
		if subtle.ConstantTimeCompare([]byte(supplied), []byte(configured)) != 1 {
			renderJSON(w, http.StatusUnauthorized, apiError{Code: http.StatusUnauthorized, Message: "invalid or missing API key"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) apiLeadSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderJSON(w, http.StatusMethodNotAllowed, apiError{Code: http.StatusMethodNotAllowed, Message: methodNotAllowedMessage})
		return
	}

	var req leadSearchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "invalid JSON body"})
		return
	}

	req.Business = strings.TrimSpace(req.Business)
	req.Location = strings.TrimSpace(req.Location)
	req.Lang = strings.TrimSpace(req.Lang)

	if req.Business == "" {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "business is required"})
		return
	}
	if req.Location == "" && req.GeoCoordinates == "" {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "location or geo_coordinates is required"})
		return
	}
	if req.Limit == 0 {
		req.Limit = 25
	}
	if req.Limit < 1 || req.Limit > 200 {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "limit must be between 1 and 200"})
		return
	}
	if req.Depth == 0 {
		req.Depth = 3
	}
	if req.Depth < 1 || req.Depth > 100 {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "depth must be between 1 and 100"})
		return
	}
	if req.Lang == "" {
		req.Lang = "en"
	}
	if req.Radius == 0 {
		req.Radius = 10000
	}
	if req.Radius < 1 {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "radius must be positive"})
		return
	}
	if req.MaxTime == 0 {
		req.MaxTime = 300
	}
	if req.MaxTime < 30 || req.MaxTime > 900 {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "max_time must be between 30 and 900 seconds"})
		return
	}
	if req.FastMode && req.GeoCoordinates == "" {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "fast_mode requires geo_coordinates"})
		return
	}
	if req.FastMode && req.Zoom == 0 {
		req.Zoom = 15
	}
	if req.Zoom != 0 && (req.Zoom < 1 || req.Zoom > 21) {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "zoom must be between 1 and 21"})
		return
	}

	search := req.Business
	if req.Location != "" {
		search += " in " + req.Location
	}

	job := Job{
		ID:     uuid.New().String(),
		Name:   search,
		Date:   time.Now().UTC(),
		Status: StatusPending,
		Data: JobData{
			Keywords: []string{search},
			Lang: req.Lang, Zoom: req.Zoom, FastMode: req.FastMode,
			Radius: req.Radius, Depth: req.Depth, Email: req.Email,
			MaxTime: time.Duration(req.MaxTime) * time.Second,
		},
	}

	if req.GeoCoordinates != "" {
		parts := strings.Split(req.GeoCoordinates, ",")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "geo_coordinates must be lat,lon"})
			return
		}
		job.Data.Lat = strings.TrimSpace(parts[0])
		job.Data.Lon = strings.TrimSpace(parts[1])
	}

	if err := job.Validate(); err != nil {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	if err := s.svc.Create(r.Context(), &job); err != nil {
		renderJSON(w, http.StatusInternalServerError, apiError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	renderJSON(w, http.StatusAccepted, leadSearchCreated{
		ID: job.ID, Status: job.Status, Search: search,
		ResultURL: "/api/v1/lead-search/" + job.ID + "?limit=" + strconv.Itoa(req.Limit),
	})
}

func (s *Server) apiLeadSearchResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		renderJSON(w, http.StatusMethodNotAllowed, apiError{Code: http.StatusMethodNotAllowed, Message: methodNotAllowedMessage})
		return
	}

	id, ok := getIDFromRequest(r)
	if !ok {
		renderJSON(w, http.StatusBadRequest, apiError{Code: http.StatusBadRequest, Message: "invalid job id"})
		return
	}

	job, err := s.svc.Get(r.Context(), id.String())
	if err != nil {
		renderJSON(w, http.StatusNotFound, apiError{Code: http.StatusNotFound, Message: http.StatusText(http.StatusNotFound)})
		return
	}

	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	out := leadSearchResult{ID: job.ID, Status: job.Status, Search: job.Name, Requested: limit}
	if job.Status == StatusFailed {
		out.Error = "scrape job failed"
		renderJSON(w, http.StatusOK, out)
		return
	}
	if job.Status != StatusOK {
		renderJSON(w, http.StatusOK, out)
		return
	}

	path, err := s.svc.GetCSV(r.Context(), job.ID)
	if err != nil {
		renderJSON(w, http.StatusInternalServerError, apiError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	results, err := readLeadCSV(path, limit)
	if err != nil {
		renderJSON(w, http.StatusInternalServerError, apiError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	out.Count = len(results)
	out.Results = results
	renderJSON(w, http.StatusOK, out)
}

func readLeadCSV(path string, limit int) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return []map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	results := make([]map[string]string, 0, limit)
	for len(results) < limit {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		item := make(map[string]string, len(header))
		for i, key := range header {
			if i < len(row) {
				item[key] = row[i]
			}
		}
		results = append(results, item)
	}
	return results, nil
}
