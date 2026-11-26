package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	java "github.com/abxengg/codist-java"
	model "github.com/abxengg/codist-model"
	"github.com/abxengg/codist/internal/web"
)

// Chat: questions about the codebase, answered by the local model from what Codist knows. For each question Codist
// picks the classes it is about (by class, method, endpoint and file names, plus the file you are looking at), and
// gives the model their code, a map of the services and the open issues. The answer names the files it used.

type chatTurn struct {
	Role    string `json:"role"` // user or assistant
	Content string `json:"content"`
}

type chatSource struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Name string `json:"name,omitempty"`
}

var chatWord = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]{2,}`)

// chatWords splits a question into lower-case words, with camelCase names also split into their parts.
func chatWords(q string) []string {
	var out []string
	for _, w := range chatWord.FindAllString(q, -1) {
		out = append(out, strings.ToLower(w))
		var part []rune
		for i, r := range w {
			if i > 0 && unicode.IsUpper(r) && len(part) > 0 {
				if len(part) > 2 {
					out = append(out, strings.ToLower(string(part)))
				}
				part = nil
			}
			part = append(part, r)
		}
		if len(part) > 2 && len(part) < len(w) {
			out = append(out, strings.ToLower(string(part)))
		}
	}
	return out
}

var chatStop = map[string]bool{"the": true, "and": true, "for": true, "what": true, "how": true, "does": true, "this": true, "that": true,
	"with": true, "from": true, "where": true, "when": true, "which": true, "who": true, "why": true, "are": true, "is": true, "can": true,
	"code": true, "class": true, "method": true, "file": true, "about": true, "there": true, "any": true, "all": true, "show": true, "explain": true}

// chatPick ranks the classes by how well they match the question; the file on screen counts most.
func chatPick(ix *java.Index, words []string, path string) []*java.Type {
	type scored struct {
		t *java.Type
		s int
	}
	var all []scored
	for _, t := range ix.Types {
		if t.Stereotype == "test" {
			continue
		}
		s := 0
		if path != "" && t.File == path {
			s += 40
		}
		name, file := strings.ToLower(t.Name), strings.ToLower(t.File)
		for _, w := range words {
			if chatStop[w] {
				continue
			}
			switch {
			case name == w:
				s += 12
			case strings.Contains(name, w):
				s += 4
			}
			if strings.Contains(file, w) {
				s++
			}
			for _, m := range t.Methods {
				if strings.ToLower(m.Name) == w {
					s += 3
				}
			}
			for _, e := range t.Endpoints {
				if strings.Contains(strings.ToLower(e.Path), w) {
					s += 3
				}
			}
		}
		if s > 0 {
			all = append(all, scored{t, s})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].s > all[j].s })
	var out []*java.Type
	for i := 0; i < len(all) && i < 6; i++ {
		out = append(out, all[i].t)
	}
	return out
}

// chatContext is what the model is told: the services, the open issues, and the picked classes, the first three
// with their code and the rest as outlines.
func (a *App) chatContext(ix *java.Index, picked []*java.Type) string {
	var b strings.Builder
	fmt.Fprintf(&b, "REPOSITORY %s: %d Java files, %d lines, %d classes.\nSERVICES:\n", filepath.Base(a.repo.Root), ix.Files, ix.Lines, len(ix.Types))
	for _, m := range ix.Modules {
		fmt.Fprintf(&b, "- %s (%s, %d classes", m.Name, m.Kind, m.Types)
		if m.Port != "" {
			fmt.Fprintf(&b, ", port %s", m.Port)
		}
		b.WriteString(")\n")
	}
	if open := a.store.OpenIssueList(); len(open) > 0 {
		b.WriteString("OPEN ISSUES:\n")
		for i, is := range open {
			if i == 15 {
				break
			}
			fmt.Fprintf(&b, "- [%s] %s at %s:%d\n", is.Severity, is.Title, is.Path, is.Line)
		}
	}
	for i, t := range picked {
		fmt.Fprintf(&b, "\nCLASS %s (%s, %s) in %s, lines %d-%d\n", t.ID, t.Kind, t.Stereotype, t.File, t.Line, t.End)
		for _, e := range t.Endpoints {
			fmt.Fprintf(&b, "  endpoint %s %s -> %s() line %d\n", e.Verb, e.Path, e.Handler, e.Line)
		}
		src := ix.Sources[t.File]
		if i < 3 && len(src) > 0 {
			lines := strings.Split(string(src), "\n")
			from, to := max(1, t.Line), min(len(lines), t.End)
			if to-from > 140 {
				to = from + 140
			}
			for n := from; n <= to; n++ {
				fmt.Fprintf(&b, "%4d %s\n", n, lines[n-1])
			}
			continue
		}
		for _, m := range t.Methods {
			fmt.Fprintf(&b, "  line %d: %s\n", m.Line, m.Signature)
		}
	}
	return b.String()
}

const chatSystem = `You are Codist, an assistant that knows one Java codebase. Answer the developer's question from the
context given: the services, the open issues and the code of the classes that matched. Be specific: name classes,
methods and endpoints, and say which file and line. If the context does not hold the answer, say what you would
need to look at instead of guessing. Write plain, short markdown: a direct answer first, then details. Use code
blocks for code. In "sources", list the files and lines your answer relies on, from the context only.`

var chatSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answer": map[string]any{"type": "string"},
		"sources": map[string]any{"type": "array", "items": map[string]any{"type": "object",
			"properties": map[string]any{"path": map[string]any{"type": "string"}, "line": map[string]any{"type": "integer"}},
			"required":   []string{"path", "line"}}},
	},
	"required": []string{"answer", "sources"},
}

func (a *App) chat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Messages []chatTurn `json:"messages"`
		Path     string     `json:"path"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.Messages) == 0 || in.Messages[len(in.Messages)-1].Role != "user" {
		web.Error(w, http.StatusBadRequest, "send the conversation, ending with your question")
		return
	}
	a.mu.RLock()
	var m *model.Client
	if a.modelOK && !a.cfg.ModelOff {
		m = a.model
	}
	a.mu.RUnlock()
	if m == nil {
		web.Error(w, http.StatusServiceUnavailable, "The AI model is not available, so chat is off. Codist's code checks still run.")
		return
	}
	// The words of the whole conversation pick the classes, the latest question counting twice.
	q := in.Messages[len(in.Messages)-1].Content
	var said strings.Builder
	for _, t := range in.Messages {
		if t.Role == "user" {
			said.WriteString(t.Content + " ")
		}
	}
	a.chats.Add(1) // the model lane holds back new reviews until this answer is done
	defer a.chats.Add(-1)
	ix := a.index()
	picked := chatPick(ix, append(chatWords(said.String()), chatWords(q)...), in.Path)

	var user strings.Builder
	user.WriteString(a.chatContext(ix, picked))
	user.WriteString("\nCONVERSATION:\n")
	turns := in.Messages
	if len(turns) > 8 {
		turns = turns[len(turns)-8:]
	}
	for _, t := range turns {
		fmt.Fprintf(&user, "%s: %s\n", strings.ToUpper(t.Role), t.Content)
	}
	user.WriteString("\nAnswer the last USER message.")

	ctx, cancel := context.WithTimeout(model.WithLabel(r.Context(), model.Label{Kind: "chat", Title: clip(q, 60), Path: in.Path}), 5*time.Minute)
	defer cancel()
	var out struct {
		Answer  string       `json:"answer"`
		Sources []chatSource `json:"sources"`
	}
	start := time.Now()
	if _, err := m.ChatJSON(ctx, chatSystem, user.String(), chatSchema, &out); err != nil {
		web.Error(w, http.StatusBadGateway, "The model did not answer: "+err.Error())
		return
	}
	// Keep only sources that are real files, and say which classes were looked at.
	var srcs []chatSource
	for _, s := range out.Sources {
		if _, ok := ix.Sources[s.Path]; ok {
			srcs = append(srcs, s)
		}
	}
	var looked []chatSource
	for _, t := range picked {
		looked = append(looked, chatSource{Path: t.File, Line: t.Line, Name: t.Name})
	}
	web.JSON(w, map[string]any{"answer": out.Answer, "sources": srcs, "looked": looked, "took": time.Since(start).Milliseconds(), "model": m.Name})
}

// clip cuts s to n characters, ending it with an ellipsis when it was longer.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
