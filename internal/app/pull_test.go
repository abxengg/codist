package app

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	git "github.com/abxengg/codist-git"
	"github.com/abxengg/codist-git/testrepo"
	review "github.com/abxengg/codist-review"
	"github.com/abxengg/codist/internal/config"
)

// An Ollama that does not have the model yet: the status offers the download, and after it the model is ready.
func TestPullDownloadsTheMissingModel(t *testing.T) {
	var have atomic.Bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			if have.Load() {
				io.WriteString(w, `{"models":[{"name":"small:latest"}]}`)
			} else {
				io.WriteString(w, `{"models":[{"name":"other"}]}`)
			}
		case "/api/pull":
			io.WriteString(w, `{"status":"pulling manifest"}`+"\n")
			for i := int64(1); i <= 3; i++ {
				fmt.Fprintf(w, `{"status":"downloading","completed":%d,"total":3}`+"\n", i)
			}
			have.Store(true)
			io.WriteString(w, `{"status":"success"}`+"\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()
	e := start(t, func(c *config.Config) { c.ModelURL, c.Model = ollama.URL, "small" })

	var st struct {
		CanPull    bool   `json:"canPull"`
		LLMProblem string `json:"llmProblem"`
	}
	e.get("/api/status", &st)
	if !st.CanPull || st.LLMProblem == "" {
		t.Fatalf("a missing model should offer the download: %+v", st)
	}
	if code := e.call(http.MethodPost, "/api/model/pull", "", nil); code != http.StatusOK {
		t.Fatalf("pull answered %d", code)
	}
	waitFor(t, "the download to finish", func() bool {
		var p pullState
		e.get("/api/model/pull", &p)
		return p.Done
	})
	e.get("/api/status", &st)
	if st.CanPull || st.LLMProblem != "" {
		t.Fatalf("after the download the model should be ready: %+v", st)
	}
	// Nothing left to download: a second request is refused.
	if code := e.call(http.MethodPost, "/api/model/pull", "", nil); code != http.StatusConflict {
		t.Fatalf("a second pull answered %d", code)
	}
}

// With no server at all a download cannot help, so none is offered.
func TestPullNotOfferedWithoutOllama(t *testing.T) {
	e := start(t, func(c *config.Config) { c.ModelURL = "http://127.0.0.1:1" })
	var st struct {
		CanPull bool `json:"canPull"`
	}
	e.get("/api/status", &st)
	if st.CanPull {
		t.Fatal("no server, no download")
	}
}

func TestColdPick(t *testing.T) {
	at := time.Unix(0, 0)
	c := func(sha string, parents int, files ...git.FileStat) git.CommitStat {
		return git.CommitStat{SHA: sha, Time: at, Parents: parents, Files: files}
	}
	f := func(p string, add, del int) git.FileStat { return git.FileStat{Path: p, Added: add, Deleted: del} }
	hist := []git.CommitStat{ // oldest first
		c("old", 1, f("src/main/java/Order.java", 30, 0)),       // too old
		c("tweak-old", 1, f("src/main/java/Order.java", 3, 2)),  // too old
		c("docs", 1, f("README.md", 40, 12)),                    // no Java
		c("tests", 1, f("src/test/java/OrderTest.java", 20, 9)), // tests only
		c("format", 1, f("src/main/java/Order.java", 900, 900)), // too big
		c("merge", 2, f("src/main/java/Order.java", 5, 3)),      // a merge
		c("feature", 1, f("src/main/java/Refund.java", 220, 0)), // only adds
		c("edit", 1, f("src/main/java/Order.java", 4, 3)),       // a small change to a file changed often
		c("edit-new", 1, f("src/main/java/Refund.java", 6, 2)),  // a change to a file changed once
	}
	if got := coldPicks(hist, 7, 3); !slices.Equal(got, []string{"edit", "edit-new"}) {
		t.Fatalf("coldPicks = %q, want edit then edit-new: small changes to code that was there, the file changed most often first", got)
	}
	if got := coldPicks(hist, 7, 1); !slices.Equal(got, []string{"edit"}) {
		t.Fatalf("coldPicks = %q, want only edit", got)
	}
	if got := coldPicks(hist[:7], 5, 3); len(got) != 0 {
		t.Fatalf("coldPicks = %q, want none: nothing recent changes code that was there", got)
	}
}

// A commit the model has read is not checked again by the fast lane: that would replace the model's review with one
// without its findings. On a first run the riskiest commits reach the model before the checks lane gets to them.
func TestChecksLaneKeepsTheModelsReview(t *testing.T) {
	e := start(t, nil)
	e.model.mu.Lock()
	e.model.review = `{"findings":[{"line":` + fmt.Sprint(testrepo.BugLine) + `,"severity":"high","title":"Discount applied twice","evidence":"total = total * 9 / 10;","detail":"d"}]}`
	e.model.mu.Unlock()
	read := func() string { // the model's name once it has finished reading the commit
		if rv, _ := e.app.store.GetReview(e.repo.Bug); rv.Status == "done" {
			return rv.Model
		}
		return ""
	}
	waitFor(t, "the model to read the bug commit", func() bool { return read() != "" })
	before := len(e.app.store.Findings(e.repo.Bug))
	e.app.checks.Push(job{sha: e.repo.Bug, opt: review.Options{RulesOnly: true}}, true)
	waitFor(t, "the checks lane to take the job", func() bool { return e.app.checks.Length() == 0 && !e.app.checks.Running(e.repo.Bug) })
	time.Sleep(200 * time.Millisecond)
	if read() == "" || len(e.app.store.Findings(e.repo.Bug)) != before {
		t.Fatalf("the checks lane replaced the model's review: model %q, findings %d, were %d", read(), len(e.app.store.Findings(e.repo.Bug)), before)
	}
}
