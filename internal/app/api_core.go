package app

import (
	"cmp"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	review "github.com/abxengg/codist-review"
	"github.com/abxengg/codist/internal/web"
)

func (a *App) status(w http.ResponseWriter, r *http.Request) {
	ix := a.index()
	a.mu.RLock()
	modelOK, note, view := a.modelOK, a.modelNote, a.view
	a.mu.RUnlock()
	checkedOut := a.repo.Branch()
	branch := cmp.Or(view, checkedOut)
	problem := ""
	if a.cfg.ModelOff {
		problem = "Model review is off (CODIST_MODEL_REVIEW)."
	} else if !modelOK {
		problem = note
	}
	head := a.repo.Resolve(a.viewRef())
	web.JSON(w, map[string]any{
		"repo": filepath.Base(a.repo.Root), "path": a.repo.Root, "branch": branch, "viewing": view != "", "checkedOut": checkedOut,
		"tracking": branch, "head": head, "model": a.cfg.Model, "llmProblem": problem,
		"graphSource": head, "graphBuiltAt": time.Now().UnixMilli(), "canFix": !a.cfg.ModelOff && modelOK && view == "", "user": "",
		"rules": 15, "models": map[string]string{"review": a.cfg.Model, "summary": a.cfg.Model, "fix": a.cfg.Model, "ask": a.cfg.Model},
		"types": len(ix.Types), "behaviour": a.runner != nil, "modelUsage": review.Usage(a.store), "riskModel": a.riskModel(),
	})
}

func (a *App) refs(w http.ResponseWriter, r *http.Request) {
	current := a.repo.Branch()
	out := []map[string]any{}
	for _, ref := range a.repo.Refs() {
		out = append(out, map[string]any{"name": ref.Name, "remote": ref.Remote, "sha": ref.SHA, "message": ref.Message,
			"by": ref.By, "at": ms(ref.At), "checkedOut": !ref.Remote && ref.Name == current})
	}
	web.JSON(w, out)
}

// follow switches the branch Codist shows. Another branch is read from git, so nothing in the
// working tree changes.
func (a *App) follow(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		web.Error(w, http.StatusBadRequest, "Name a branch.")
		return
	}
	if a.repo.Resolve(in.Name) == "" {
		web.Error(w, http.StatusNotFound, "There is no branch "+in.Name+".")
		return
	}
	a.mu.Lock()
	a.view = in.Name
	if in.Name == a.repo.Branch() {
		a.view = ""
	}
	a.mu.Unlock()
	go func() {
		a.index()
		a.events.publish("branch", map[string]string{"branch": in.Name})
	}()
	web.JSON(w, map[string]string{"branch": in.Name})
}

func (a *App) file(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	src, ok := a.source(q.Get("path"), q.Get("sha"))
	if !ok {
		web.Error(w, http.StatusNotFound, "That file is not there.")
		return
	}
	web.JSON(w, map[string]string{"content": string(src)})
}
