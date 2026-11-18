# How Codist handles each git operation

Codist learns everything through git and only reads from it: it never commits, pushes, pulls, merges or rewrites
anything, except a fix you approved. This page says what happens for each common git operation. **Tested** means a
test or a real run covers it; **from the code** means it was worked out by reading the code and has no test yet.

## Everyday commands

| You do this | What Codist does | Status |
|---|---|---|
| `git commit` | Sees the new commit within about 3 seconds and puts it at the front of every lane. | Tested |
| `git pull`, or a fetch that brings many commits | Queues every commit between where you were and where you are, newest first. | Tested |
| `git merge`, no conflicts | A clean merge has nothing a person wrote, so nothing to review; the merged commits are reviewed one by one. | Tested |
| `git merge`, conflicts settled by hand | Reviews exactly the lines written to settle the conflict (`git show --remerge-diff`). Needs git 2.36 or newer; with an older git a merge looks empty. | Tested |
| `git switch` to another branch | Queues the commits of that branch it has not reviewed. The branch picker on the page shows any branch without switching. | From the code |
| `git revert` | Reviewed like any commit; issues on the lines it removes are closed by it. The risk model counts the reverted commit as one that brought a bug. | From the code |
| `git cherry-pick` | A new commit with a new id, so it is reviewed. A model question asked before word for word is answered from the cache. | From the code |
| `git rm`, or deleting a file | Every open issue in the file is closed by the commit that deleted it. | From the code |
| Uncommitted edits, `git add`, `git stash` | Nothing: Codist reviews commits. | By design |

## Rewriting history

| You do this | What Codist does | Status |
|---|---|---|
| `git commit --amend`, `git rebase`, squash | The new commits are reviewed. An issue the rewritten commit still has is filed under it, so a later fix closes it. Reviews of the old ids stay in the database and are no longer shown. | Tested (v0.9.0) |
| `git reset --hard` to an older commit | Nothing new to review. Issues of the commits thrown away are closed by the next commit that changes their file, since their lines are gone. | Tested (v0.9.0) |
| A colleague force-pushed and you pulled | As a rebase. If the commit Codist last stood on is gone, it looks at the 50 newest commits and reviews those it has not seen. | From the code |

An issue first seen on a commit that still exists on **another branch** is followed on the branch you are on only
([codist-review#7](https://github.com/abxengg/codist-review/issues/7)).

## Files

| The situation | What Codist does | Status |
|---|---|---|
| A file renamed or moved (`git mv`) | Open issues whose line is still in the file move to the new name, with your judgements; the others are closed. Also when catching up meets the rename before the commit that brought the issue. | Tested (v0.9.0) |
| A commit changes more than `max_files` files | The model decides which to read, with a reason for each; you can have any skipped file read. Code checks read every file. | Tried once |
| Not Java: SQL, YAML, JavaScript | Not checked. Text documents are read only to compare a changed number with the number they state. | By design |
| Generated code, build output | `target/`, `build/`, `node_modules/`, `.idea/` are skipped; add more under `ignore`. | Setting |
| Only comments, blank lines or tests changed | Code checks run; the model skips the file. | Tested |
| Shallow clones, submodules, Git LFS | Not tested. In a shallow clone only the commits present are reviewed. | Not tested |

## First run and old history

- **First run.** Codist checks every Java file as it is today (the baseline) and files each problem under the commit
  that last changed its line (`git blame`). Deleted files are not part of today's code, so they are never scanned.
- **Old commits.** Code checks go through the whole history, about 40 ms a commit, newest first. A problem found in
  an old commit that a later commit already fixed (or whose file was deleted) is recorded as arrived and fixed, and
  never shows as open.
- **The model** reads the last `history` commits (50 by default). With `older = true` it goes back further whenever
  it has nothing else to read, one commit at a time.

## Commit messages

- A message is what the author wanted, not proof the change is right: your rules in `.codist/rules.md` outrank it.
- A message that says the commit only tidies ("refactor", "rename", "clean up") and code whose results changed gives
  a `behaviour-change` issue, when behaviour runs are on.
- Text in a message or in the code that tries to instruct an AI reviewer is reviewed, never obeyed, and reported.
- Words like "fix" and "revert" in old messages teach the risk model which commits brought bugs.

## When git is the problem

| | What Codist does |
|---|---|
| git not installed | Says so in one line and stops. |
| Not inside a git repository | Says so in one line and stops. |
| A commit git cannot show | The review is marked failed and tried again on the next start. |
