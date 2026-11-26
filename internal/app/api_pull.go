package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/abxengg/codist/internal/web"
)

// Downloading the model from the interface: when Ollama runs but does not have the model, the page offers to fetch
// it and shows how far along it is, instead of only saying the AI is offline. Ollama does the download; Codist asks
// it to and passes its progress on.

// pullState is the download as the page sees it.
type pullState struct {
	Running   bool   `json:"running"`
	Status    string `json:"status"` // Ollama's words: "pulling manifest", "downloading", "verifying sha256 digest", "success"
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
	Error     string `json:"error,omitempty"`
	Done      bool   `json:"done"`
}

var (
	pullMu  sync.Mutex
	pulling pullState
)

// canPull says the model server is Ollama and lacks the model, which is the one case a download fixes.
func canPull(note string) bool { return strings.Contains(note, "ollama pull") }

func (a *App) pullStatus(w http.ResponseWriter, r *http.Request) {
	pullMu.Lock()
	defer pullMu.Unlock()
	web.JSON(w, pulling)
}

func (a *App) pull(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	note := a.modelNote
	a.mu.RUnlock()
	pullMu.Lock()
	defer pullMu.Unlock()
	switch {
	case pulling.Running:
		web.JSON(w, pulling)
		return
	case !canPull(note):
		web.Error(w, http.StatusConflict, "There is nothing to download: "+note)
		return
	}
	pulling = pullState{Running: true, Status: "starting"}
	go a.runPull()
	web.JSON(w, pulling)
}

// runPull asks Ollama for the model and follows its progress lines until it says success or fails.
func (a *App) runPull() {
	set := func(f func(*pullState)) {
		pullMu.Lock()
		f(&pulling)
		s := pulling
		pullMu.Unlock()
		a.events.publish("pull", s)
	}
	body, _ := json.Marshal(map[string]any{"model": a.cfg.Model, "stream": true})
	req, _ := http.NewRequestWithContext(a.ctx, http.MethodPost, strings.TrimSuffix(a.cfg.ModelURL, "/")+"/api/pull", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	slog.Info("model download started", "model", a.cfg.Model)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		set(func(p *pullState) { p.Running, p.Error = false, "Ollama did not answer: "+err.Error() })
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		set(func(p *pullState) { p.Running, p.Error = false, "Ollama said "+resp.Status })
		return
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	shown := time.Time{}
	for sc.Scan() {
		var line struct {
			Status    string `json:"status"`
			Completed int64  `json:"completed"`
			Total     int64  `json:"total"`
			Error     string `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Error != "" {
			set(func(p *pullState) { p.Running, p.Error = false, line.Error })
			slog.Warn("model download failed", "model", a.cfg.Model, "err", line.Error)
			return
		}
		if line.Status == "success" {
			break
		}
		// Progress comes many times a second; the page needs a few.
		if line.Total > 0 && time.Since(shown) < 400*time.Millisecond {
			continue
		}
		shown = time.Now()
		set(func(p *pullState) {
			p.Status = line.Status
			if line.Total > 0 {
				p.Completed, p.Total = line.Completed, line.Total
			}
		})
	}
	if err := sc.Err(); err != nil {
		set(func(p *pullState) { p.Running, p.Error = false, "The download stopped: "+err.Error() })
		return
	}
	ok, note := a.model.Status(a.ctx)
	a.mu.Lock()
	a.modelOK, a.modelNote = ok, note
	a.mu.Unlock()
	slog.Info("model download finished", "model", a.cfg.Model, "ready", ok, "took", time.Since(start).Round(time.Second))
	set(func(p *pullState) { p.Running, p.Done, p.Status = false, true, "success" })
	a.events.publish("status", map[string]any{"llmProblem": ""})
}
