package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// version is set by the release build (-ldflags "-X main.version=v0.11.0"). A build from `go install …@v0.11.0`
// reads it from the module instead; any other build is "dev".
var version = ""

const releases = "https://github.com/abxengg/codist/releases"

func currentVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

// assetName is the release file for this system, as `make dist` names them.
func assetName(goos, goarch string) (string, error) {
	osName := map[string]string{"darwin": "macos", "linux": "linux", "windows": "windows"}[goos]
	arch := map[string]string{"arm64": "arm64", "amd64": "x64"}[goarch]
	if osName == "" || arch == "" || (osName == "windows" && arch != "x64") {
		return "", fmt.Errorf("there is no release build for %s/%s; build from source with CODIST_FROM_SOURCE=1 sh install.sh", goos, goarch)
	}
	name := "codist-" + osName + "-" + arch
	if osName == "windows" {
		name += ".exe"
	}
	return name, nil
}

// newer says whether release tag a is newer than b, comparing vMAJOR.MINOR.PATCH by number. A "dev" build is older
// than any release.
func newer(a, b string) bool {
	parse := func(v string) ([3]int, bool) {
		var out [3]int
		parts := strings.SplitN(strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v"), ".", 3)
		if len(parts) != 3 {
			return out, false
		}
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				return out, false
			}
			out[i] = n
		}
		return out, true
	}
	x, okA := parse(a)
	y, okB := parse(b)
	if !okA {
		return false
	}
	if !okB {
		return true
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// latestRelease asks GitHub for the newest release's tag. Nothing about you or your code is sent.
func latestRelease() (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos/abxengg/codist/releases/latest", nil)
	req.Header.Set("User-Agent", "codist")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var r struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || r.Tag == "" {
		return "", errors.New("GitHub's answer had no release in it")
	}
	return r.Tag, nil
}

// runVersion prints the version and, unless told not to, whether a newer release exists.
func runVersion(args []string) {
	cur := currentVersion()
	fmt.Printf("codist %s (%s/%s)\n", cur, runtime.GOOS, runtime.GOARCH)
	if len(args) > 0 && args[0] == "--no-check" {
		return
	}
	latest, err := latestRelease()
	switch {
	case err != nil:
		fmt.Println("Could not check for a newer release:", err)
	case newer(latest, cur):
		fmt.Printf("A newer release exists: %s. Run: codist upgrade\n", latest)
	default:
		fmt.Println("This is the latest release.")
	}
}

// runUpgrade replaces this binary with a release: the latest, or the tag given. The new binary is downloaded next to
// this one, run once to check it works, and only then swapped in; the old one is kept as codist.previous.
func runUpgrade(args []string) error {
	cur := currentVersion()
	want := ""
	if len(args) > 0 {
		want = args[0]
		if !strings.HasPrefix(want, "v") {
			want = "v" + want
		}
	} else {
		latest, err := latestRelease()
		if err != nil {
			return fmt.Errorf("could not find the latest release: %w", err)
		}
		if !newer(latest, cur) {
			fmt.Printf("codist %s is the latest release.\n", cur)
			return nil
		}
		want = latest
	}
	if want == cur {
		fmt.Printf("codist %s is already installed.\n", cur)
		return nil
	}
	asset, err := assetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if self, err = filepath.EvalSymlinks(self); err != nil {
		return err
	}
	dir := filepath.Dir(self)
	tmp := filepath.Join(dir, ".codist-"+want+".download")
	url := releases + "/download/" + want + "/" + asset
	fmt.Printf("Downloading %s (%s)...\n", asset, want)
	if err := download(url, tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("download %s: %w", url, err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	if runtime.GOOS == "darwin" { // a downloaded file is marked; the release is not notarised, so macOS would block it
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", tmp).Run()
	}
	// Run it once before swapping it in: "help" is in every release, old and new.
	out, err := exec.Command(tmp, "help").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "codist doctor") {
		os.Remove(tmp)
		return fmt.Errorf("the downloaded codist does not run (%v): %s", err, strings.TrimSpace(string(out)))
	}
	prev := filepath.Join(dir, "codist.previous")
	if runtime.GOOS == "windows" {
		prev += ".exe"
	}
	_ = os.Remove(prev)
	if err := os.Rename(self, prev); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("could not move the old codist aside (is %s writable?): %w", dir, err)
	}
	if err := os.Rename(tmp, self); err != nil {
		_ = os.Rename(prev, self)
		os.Remove(tmp)
		return fmt.Errorf("could not put the new codist in place: %w", err)
	}
	verb := "Upgraded"
	if newer(cur, want) {
		verb = "Went back"
	}
	fmt.Printf("%s: %s -> %s (%s)\nThe old one is kept as %s.\n", verb, cur, want, self, prev)
	return nil
}

func download(url, to string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "codist")
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	f, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
