package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/guil/lumo-api/internal/browser"
)

type PromptRequest struct {
	Prompt    string `json:"prompt"`
	WebSearch bool   `json:"webSearch,omitempty"`
	Debug     bool   `json:"debug,omitempty"`   // reserved for future use
	Timeout   int    `json:"timeout,omitempty"` // seconds, defaults to 30
}

type PromptResponse struct {
	Answer string `json:"answer,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewHandler returns an http.Handler that routes /v1/prompt.
func NewHandler(drv *browser.Driver) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/prompt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(req.Prompt) == 0 || len(req.Prompt) > 4096 {
			http.Error(w, "prompt length invalid", http.StatusBadRequest)
			return
		}
		// Default timeout 30 s if not supplied.
		to := 30 * time.Second
		if req.Timeout > 0 {
			to = time.Duration(req.Timeout) * time.Second
		}
		ctx, cancel := time.WithTimeout(r.Context(), to)
		defer cancel()

		ans, err := drv.Prompt(ctx, req.Prompt, req.WebSearch)
		resp := PromptResponse{}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Answer = ans
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}
