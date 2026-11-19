# Getting started

This takes about ten minutes, most of it downloading a model.

## 1. Install Codist

**macOS and Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/abxengg/codist/remaster/install.sh | sh
```

**Windows**, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/abxengg/codist/remaster/install.ps1 | iex
```

The script puts one file, `codist` (`codist.exe` on Windows), in `~/.local/bin` (`%LOCALAPPDATA%\codist\bin` on
Windows), and then checks what Codist needs:

```
Checking what Codist needs:
  ok       git 2.54.0
  ok       java (optional; for checks that run code)
  -        no model server at http://localhost:11434 (optional; Codist still runs its code checks)
```

If it says to add a folder to your `PATH`, do that and open a new terminal.

**Docker**: the image has git, Java 21 and Maven, so checks that run code and fixes work without installing
anything else:

```bash
cd ~/dev/your-spring-app
docker run --rm -p 8080:8080 -v "$PWD:/repo" -v codist-data:/data \
  --add-host=host.docker.internal:host-gateway ghcr.io/abxengg/codist
```

Then open http://localhost:8080. The container reaches Ollama on your machine at `host.docker.internal:11434`; set
`-e CODIST_MODEL_URL=...` for another address. To open it from another machine by name, add `-e CODIST_HOSTS=that-name`. Pin a version with `ghcr.io/abxengg/codist:0.5.1`.

**With Go** 1.26 or later:

```bash
go install github.com/abxengg/codist@latest
```

With the install script, you can choose a version or a folder:

```bash
CODIST_VERSION=v0.10.0 CODIST_BIN_DIR=/usr/local/bin sh install.sh
CODIST_FROM_SOURCE=1 sh install.sh      # build with Go instead of downloading
```

### What Codist needs

| | Needed? | Why |
|---|---|---|
| git | yes | Codist reads your commits through it |
| a model server | no | for the AI review and incident causes; code checks work without it |
| Java 21 | no | only for checks that run code |
| Node, Go, a C compiler | no | only to work on Codist itself |

## 2. Add a model (optional, recommended)

Codist does not contain an AI model. You run a model server and Codist connects to it. The simplest is
[Ollama](https://ollama.com):

```bash
ollama pull qwen3.5:9b           # about 8 GB, the default; wants 16 GB of memory
ollama pull qwen2.5-coder:7b     # about 5 GB, for 8 GB of memory: faster, finds less
```

Ollama listens on `http://localhost:11434`, which is where Codist looks. For another address or model, see
[Settings](#4-settings). Any model Ollama has will do; set `model.name` (or `CODIST_MODEL`) to its name exactly as
`ollama list` shows it. Every measurement in these pages is for `qwen3.5:9b`. Smaller models (3b) fit in 8 GB and have
not been measured: expect more claims that Codist has to throw away, and fewer findings.

**Other model servers.** Codist speaks Ollama's API (`/api/tags`, `/api/chat`). A server that speaks only the
OpenAI-style API, such as LM Studio or llama.cpp's server, answers `unexpected endpoint` and Codist runs code checks
only ([codist-model#1](https://github.com/abxengg/codist-model/issues/1)). To check a server:

```bash
curl http://localhost:PORT/api/tags
```

A list of models means Codist can use it (`CODIST_MODEL_URL=http://localhost:PORT`). Ollama can run next to such
a server; it keeps its own copy of each model.

On a laptop the model takes minutes per commit, so Codist reads the newest commits first and, by default, only the
last 50. The quick code checks still cover every commit.

## 3. Run it on a repository

```bash
cd ~/dev/your-spring-app
codist
```

Your browser opens `http://localhost:8080`. The first time, Codist:

1. reads all the Java code (about a second for a thousand files);
2. checks the whole code once, so problems already there show up, each credited to the commit that last changed
   its line;
3. runs the code checks on every commit, which takes seconds;
4. lets the model read recent commits in the background, newest first. New commits jump the queue.

Leave it running while you work: each new commit is checked as soon as it is made. Stopping and starting again picks
up where it left off.

To look at another repository, run `codist /path/to/repo`. `codist --no-open` skips opening the browser.

## 4. Settings

Put settings in `.codist/config.lua` in the repository. All are optional:

```lua
model = { url = "http://localhost:11434", name = "qwen3.5:9b", context = 8192 }
max_files = 8            -- above this many changed files the model first decides which ones need reading
history = 50             -- recent commits the model reads in the background; 0 for all
ignore = { "target/", "build/", "generated/" }
behaviour = false        -- run changed methods before and after each recent commit (see below)
deep = false             -- when the model is idle, read recent commits again through specialist lenses
older = false            -- when the model is idle, also read the commits older than `history`, newest first
```

The file is Lua, so a team can compute values instead of copying them; it runs with no file, network or OS access.

The file comes with the repository, so two settings in it wait for your word: `behaviour = true`, which runs the
repository's code, and a `model.url` on another machine, which sends your code there. Codist says so when it
starts. To agree for a repository, run `codist trust` in it once; Codist remembers it in its data folder. The
environment variables below are yours, so they need no trust.
Environment variables override it for one run:

| Variable | Meaning |
|---|---|
| `PORT` | the port to serve on (default 8080) |
| `CODIST_MODEL`, `CODIST_MODEL_URL` | model name and server address |
| `CODIST_MODEL_REVIEW=false` | code checks only |
| `CODIST_HISTORY` | how many recent commits the model reads; 0 for all |
| `CODIST_BEHAVIOUR=1` | turn on running changed methods (see below) |
| `CODIST_DEEP=1` | read recent commits again, more deeply, whenever the model has nothing else to do |
| `CODIST_OLDER=1` | when the model has nothing else to do, read the commits older than `history` too, newest first |
| `CODIST_DATA_DIR` | where Codist keeps its data (default `~/.codist`) |
| `CODIST_HOSTS` | extra host names the pages may reach Codist by, separated by commas (see below) |
| `CODIST_LOG` | how much it logs: `debug`, `info` (default), `warn` or `error` |
| `CODIST_LOG_JSON=1` | log one JSON object per line, for log tools |

Codist has no login, so it only answers pages opened as `localhost` or `127.0.0.1`, and refuses requests that come
through another website. To open it by another name, such as a machine on your network, name it:
`CODIST_HOSTS=devbox.local codist`. This setting is read from the environment only, never from the repository.

### Your team's rules

Write the rules your code must keep in `.codist/rules.md`, in plain words: "a coupon and a loyalty discount never
stack", "cancelling releases only what was reserved". The model reads them with every review and reports a change
that breaks one, naming the rule. Keep them short; the first 4,000 characters are used. Codist reads the file fresh
for each review, so edits apply to the next commit it reads.

## 5. Using it

- **Issues** lists every open issue. **Filter** narrows it by type, urgency, change and **Found by**: the AI model or
  the code checks. An issue that both reported counts under both. Issues found by the model carry an **AI** tag.
- **Review** lists commits newest first. Pick one to see its diff with issues next to the lines. Issues from code
  checks are exact; issues from the model show the line it quoted, and claims Codist could not confirm are shown as
  notes, as is anything the model itself rates minor. A merge commit is reviewed on what was written by hand to
  settle its conflicts. Mark an issue resolved by hand, or let the commit that fixes it close it.
- **How far to believe a finding**: each one is tagged **proven** (an exact check found it, or running the code
  showed it), **likely** (the model found it, quoted the code, and the claim passed Codist's checks) or **to judge**
  (Codist could not back it, the model rated it minor, or a run showed a change the commit message does not
  explain). Only proven and likely findings open issues.
- **Older commits** (`older = true`): the model normally reads only the last `history` commits. With this on, whenever it has nothing else to read it goes back past them, one commit at a time, newest first. A new commit always goes first. On a laptop this is about a minute per commit, so a long history takes days of idle time; a restart carries on where it stopped. Not measured yet.
- **Deep pass** (`deep = true`): when the model has nothing else to read, it reads recent commits again, once as a
  specialist in failures (nulls, exceptions, loops) and once as a specialist in your team's rules, and adds what
  only those readings find. New commits always come first. It uses the machine while you are not waiting for it.
- **Files**: every changed file of a commit is considered. Code checks read all of them. For the model, each file
  is either read or skipped with a reason: tests and files where only comments changed are skipped by Codist; when
  a commit changes more than `max_files` files, the model first looks at a short summary of each change and says
  which ones a reviewer has to read (a changed condition or query, yes; a renamed field or a new getter, no; when
  unsure, read). The commit's **Files** card shows each decision, its reason and who made it, and **Check this
  file** has the model read a skipped file after all. Your request is remembered for later reviews of that commit.
- **Fix**: on a real bug, press **Fix** to get a prepared change with the reason, the current and proposed code side by
  side, and whether it still builds and passes the tests. **Apply & commit** commits just that fix; **Suggest how to
  fix it…** asks for it your way; **Reject…** with a reason is remembered for later fixes.
- **What the model did, on each commit**: every finding the model kept or demoted has **How Codist decided**, the
  checks it went through in order (the quote, exact checks, the second look with the line it quoted as proof, the
  choice between the claim and its opposite, the learned filter). The **Model** card shows what the model
  understood in each file it read (what changed, who sees a different result, what your rules say it must be) and
  every claim Codist threw away, with why: a quote not in the code, a line a check already reports, a repeat.
- **Checks learned from your fixes**: when the model reads a commit that fixes something, it writes a pattern for
  the mistake as it was written. Codist keeps the pattern only if it matches the old line, not the fixed one, and
  almost nothing else in your code; from then on it runs with the code checks, without the model, and a finding
  from it says which fix it was learned from. Measured on planted bugs it rarely
  generalises (1 of 42 caught, no false matches): expect it to catch the same mistake coming back, not its relatives.
- **Changed promises across files** (only in the deep pass, `deep = true`): when a commit changes a method that other files call, the model says what the
  method promised before and after (can it return null, what can it throw). If the promise got weaker, the finding
  names the callers in other files, and marks those that use the result at once.
- **`codist top`**, in a second terminal while Codist runs: a live view of the lanes, the model call under way (its
  stage, whether it is still reading the prompt or writing, tokens in and out, speed, the last lines it wrote),
  totals, recent calls and the log. `codist top http://host:port` watches a Codist on another address. Press q.
- **Activity** shows the three lanes (code checks, AI model, behaviour runs) with what runs now, what waits and what
  finished, every model call with the exact prompt and reply, and the live log. Use it to see why a review is slow.
- **Graph** draws services, then classes, then methods, with Feign calls between services. Pick a commit to see
  what it changed and what depends on that. Right-click a class for its blast radius.
- **Incidents**: paste logs from a failing service. Codist finds each Java error, the line in your code, the commit
  that last changed it, and a short list of suspect commits; with a model it suggests causes and checks every line
  it quotes.
- **Repository**: modules and their ports, routes and who calls them, hotspots that change often and have open
  issues, and methods that may be slow. **What became of the findings** shows, for each code check and for the
  model, how many issues it raised, how many a later commit fixed and how you judged them. A check nobody acts on
  is noise; this table is where you see it.
- **Numbers the documents state**: when a commit changes one number on a line (`DIM_EVERY = 20.0` to `10.0`) and a
  Markdown, AsciiDoc or text file in the repository still states the old number about the same thing, Codist reports
  that the two disagree and quotes the document. It needs no model. Changing the document in the same commit is
  taken as a decision and not reported; putting the value back closes the issue.

- **Behaviour**, when turned on: for each recent commit, Codist builds the code before and after it, calls the
  methods it changed (and the methods that call them) with the same inputs, and lists every call whose result
  changed: "before: 1200, after: 1201". You decide whether each change was meant. With a model, each difference
  also says whether the commit message covers it, and the card counts those it does not. That is the model's
  reading and is marked as such; no difference is ever hidden because of it. A commit that only adds code, or that
  it cannot build, says why it was not run.
  A commit whose message says it only tidies (simplify, refactor, rename, clean up, extract, inline) and whose
  methods now return something else gets an issue, `behaviour-change`, with the calls that differ; it needs no
  model. So does a call that returned before the commit and now crashes, and a result that was never negative and
  now is. Any other change of results is a note beside the code for you to judge. The model, when it reads the
  commit, is shown the changed results of the file it reads as facts.

The branch picker in the top bar shows any branch, read from git, without changing your checkout.

## Behaviour: running the changed code (optional)

Turn it on per repository with `behaviour = true` in `.codist/config.lua` and `codist trust` (once, see
[Settings](#4-settings)), or `CODIST_BEHAVIOUR=1` for one run. It
needs a JDK 17 or later (`java` and `javac`) and, for Maven projects, the project's `mvnw` or `mvn`; Gradle is not
supported yet. It is off by default because it runs your project's code.

Codist confines that code where the operating system offers a way: `sandbox-exec` on macOS (built in) and `bwrap`
on Linux (the `bubblewrap` package). Confined, the code can write only in Codist's scratch folder, cannot read the
rest of your home folder (only the JDK, Maven and the Maven cache), and has no network while its methods run. The
Maven build is confined the same way but keeps the network the first time, since it may have to download what the
project needs; later builds run offline. The environment is also cut down to the path, the JDK and the locale, so
keys and passwords in it do not reach the code. On Windows, and on a Linux without `bwrap`, there is nothing to
confine with: the code runs as you, and each report says so. On Windows the code runs at low integrity, the level
browsers use for web pages: it can write only in Codist's work folder, but it can still read your files and use the
network, and the Maven build is not confined; reports say so. Ubuntu 23.10 and later block `bwrap` through
AppArmor unless it is allowed (an AppArmor profile for `bwrap`, or `sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0`);
until then Codist cannot confine there either, and says so. Turn it on there only for code you would build yourself.
Each commit is built once and the last six builds are kept, so a run of commits builds quickly.

Inputs are standard edge cases for each parameter type, so it finds differences in logic that takes plain values
best. A service that needs a database or another service is not covered yet.

## Training data from your judgements (optional)

`codist training > findings.jsonl` writes one example for each model finding whose outcome is known: marked useful,
marked wrong, or fixed by a later commit. Each line is a chat (`messages` with system, user and assistant) that
ends in the word `useful` or `wrong`, which is the format MLX, Unsloth and llama.cpp's finetune read. Codist does not
train anything. An adapter made from this file can be put on the Ollama model with `ADAPTER` in a Modelfile. A few
hundred examples are a sensible minimum; the command says how many there are. Nothing leaves the machine.

## Updating

Run the install command again: it fetches the latest release and replaces the file. Your reviews (`~/.codist`) and
the project's settings (`.codist/config.lua`) are kept. For a given version:

```bash
curl -fsSL https://raw.githubusercontent.com/abxengg/codist/remaster/install.sh | CODIST_VERSION=v0.10.0 sh
```

Releases and what changed in each: https://github.com/abxengg/codist/releases. Codist does not yet print its version
or check for a newer one ([#14](https://github.com/abxengg/codist/issues/14)).

## Common problems

**"no model server at http://localhost:11434"**: start Ollama (open the app, or run `ollama serve`), or set
`model.url` to where your server runs. Codist checks again every 30 seconds.

**The page says "Rules only" with LM Studio or another server running**: that server does not speak Ollama's API
(see [Other model servers](#2-add-a-model-optional-recommended)). Use Ollama.

**Every log line says `model=false`**: those lines are the code checks lane, which never uses the model. A model
review logs `model=true` and takes seconds to minutes. If none appears, the model is not reachable: the page says why.

**"qwen3.5:9b is not on the server"**: run `ollama pull qwen3.5:9b`, or set `model.name` to a model you
have (`ollama list`).

**"cannot listen on 127.0.0.1:8080"**: something else uses the port. Run `PORT=8090 codist`.

**macOS says the program cannot be opened**: the installer clears the download mark; if you copied the file by hand,
run `xattr -d com.apple.quarantine ~/.local/bin/codist`.

**"Codist only answers its own pages"**: you opened it by a name other than `localhost`, or another site tried to
call it. Open `http://localhost:8080`, or allow the name with `CODIST_HOSTS`.

**Reviews are slow**: that is the model. Use `qwen2.5-coder:7b` (faster, finds less), lower `history`, or set `max_files` smaller. Code
check results show at once either way.

**Start over**: stop Codist and delete `~/.codist/<repo>.db`.

`codist doctor` runs the same checks as the installer at any time. A missing model is reported, not an error.
