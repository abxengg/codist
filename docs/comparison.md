# v0.6.0 against the next version

Measured on 2026-11-14 on a 16 GB Mac. "Before" is the released v0.6.0 built from its tag; "after" is the working
tree of the same evening (codist-review v0.11.0 to be). Each run started from an empty data folder. The model is
`qwen2.5-coder:14b` through Ollama. A bug counts as found when an alarm, not a note, lands within 6 lines of it,
or when running the code shows a difference; a clean commit fails if it gets any alarm. The scorer and answer keys
are in [codist-eval/testbed](https://github.com/abxengg/codist-eval/tree/main/testbed).

## Code checks only, no model

| Project | Commits | Bugs found before | after | Clean commits with an alarm before | after |
|---|---|---|---|---|---|
| maze-lab | 25 | 3 of 19 | **5 of 19** | 1 of 3 | 1 of 3 |
| records-lab | 26 | 11 of 19 | 11 of 19 | 0 of 5 | 0 of 5 |
| shop-lab | 32 | 8 of 24 | 8 of 24 | 0 of 6 | 0 of 6 |

The two more on maze-lab are the changed numbers the spec states (60 to 30 steps a second, 20 to 10 seconds), found
by the new `spec-number` check. It reported nothing on the other two projects or on the 442 commits below. The one
alarm on a clean maze-lab commit is a console print that the commit really adds.

## With the model: maze-lab, 25 commits, 19 planted bugs, 3 clean commits

| | v0.6.0 | next version |
|---|---|---|
| Bugs found | 15 of 19 | **17 of 19** |
| ...of the 7 that break the written spec | 4 | **7** |
| ...of the 5 hard ones | 4 | 3 |
| Bugs that got an issue with no model involved | 3 | **8** |
| Bugs the model's own findings covered | 12 | 8 |
| Clean commits with an alarm | 2 of 3 (one high, from the model) | **1 of 3** (a low console-print check; the commit does add the print) |
| Model alarms in all | 13 | 9 |
| Notes (shown, no issue) | 6 | 6 |
| Model time for the 25 commits | 32.4 min | 34.2 min |
| Model calls | 36 | 41 (30 for reviews, 11 to hold run results against commit messages) |
| Tokens read by the model for reviews | 76,481 | 70,606 |
| Wall time until everything was read | 32.6 min | 34.3 min |

What changed, bug by bug:

- **Found now, missed before (3):** the two changed numbers (`spec-number`, no model) and the maximised window, which
  the model reported both times but v0.6.0 turned into a note.
- **Found before, missed now (1):** "Keep the lantern and stun clocks running smoothly". In v0.6.0 the model raised a
  medium finding that the second look left undecided; this time the model raised nothing. One run each, so this may
  be chance, or the new describe-first step; it is not settled.
- **Still missed (1):** "Drop a stale turn sooner when blocked".
- **Weaker than it looks (1):** "Reset the frame timer after each update" counts as found because running the code
  shows the difference, but the model's correct finding was refuted by its own second look, so the page shows two
  notes and no issue. v0.6.0 raised an issue for it.
- **The model covers fewer bugs itself (12 to 8)** because four of them are now proven without it (two tidy-only
  commits whose results changed, a new crash, and one of the changed numbers), and a model finding on a line that
  already has a proven issue is dropped as a repeat. Two are the real losses named above.
- **The high false alarm is gone.** v0.6.0 flagged a clean feature commit as "redundant key press handling" and its
  second look confirmed it. The second look now refuses "redundant" as a problem.

The model-time figures are from a first run on a quiet machine. That run exposed three faults in the new code, which
were fixed and the results rescored from the model's cached replies: run notes had silenced correct model findings on
the same line; a confirmation with its quote in the wrong field counted as unproven; and a run that finished while
the model was reading had its finding overwritten. The scores above are after those fixes; the times are before them
and do not change, since the fixes add no model calls of note.

## A commit of 13 files

One commit on shop-lab's code touching 13 files: two renames, seven added constants or trivial methods, two
comment-only changes, one test, and one changed condition (`page < 1` became `page < 0`) under the message "Tidy
names, messages and labels across the order service".

| | next version |
|---|---|
| Files Codist skipped as certain (comments only, a test) | 3 |
| Files the model's plan skipped, each with a reason ("Variable renaming without logic change") | 9 |
| Files the model read | 1: `Paging.java`, "Logic change in validation" |
| Bug found | yes: "Page 0 is allowed", high |
| Model calls and time | 3 calls, 3.5 minutes |

v0.6.0 reads the first 8 files in order and says only that the rest were left out; it was not run on this commit.
One commit proves little about how often the plan skips a file it should not.

## Planted bugs, no model

`codist-eval/mutants` planted 40 small bugs in maze-lab's code, one commit each (seed 1).

| | next version |
|---|---|
| Running the code showed a difference | 22 of 40 |
| Became an issue without the model | 12 of 40 |
| ...under a commit message that says it only tidies | 10 of the 10 that running exposed |
| ...under a neutral message | 2 of the 12 that running exposed; the other 10 are notes to judge |

v0.6.0 raises no issue from a run by design: it shows differences on the commit's card only.

## Speed of the code checks

Orbit Commerce, 442 commits, two Spring services, checks only, from an empty data folder, all three measured one
after another on the same machine while a model run was using it.

| | v0.6.0 | next version |
|---|---|---|
| Time until every commit is checked | 42.5 s | **21.9 s** |
| git commands run | 1,406 (3.2 a commit) | **591** (1.3 a commit) |
| Issues found | 12 (5 open, 7 fixed) | the same 12, fixed by the same commits |
| `spec-number` findings | none | none |

Two changes make the difference: one git call for a commit and its diff where there were two (30.1 s, 1,042 calls),
and reading HEAD from the repository's own files instead of asking git once per commit (21.9 s, 591 calls). A
profile showed 17.5 of 23 seconds inside reviews were git processes, 7 of them for HEAD alone.

**Several workers for the code checks were tried and taken out again:** four workers gave 29.1 s against 31.8 s, 8%,
with the same results. The time is in starting git processes and in the one database connection, not in work that
threads can share.

## Three models on maze-lab

Same code, same project, each from an empty data folder on a quiet machine, thinking switched off. The two 9B
models were pulled from the Ollama library on 2026-11-15. Scores are after two fixes to the wording of Codist's
second look that these runs showed were needed; times are from each model's first clean run.

| | `qwen2.5-coder:14b` | `qwen3.5:9b` | `mimo-v2.6:9b` |
|---|---|---|---|
| Bugs found | 17 of 19 | **18 of 19** | 16 of 19 |
| Hard (5) / spec (7) | 3 / 7 | 4 / 7 | 3 / 6 |
| Clean commits with an alarm | 1 of 3 | 1 of 3 | 1 of 3 |
| Model alarms | 8 | 14 | 7 |
| Model time | 40.3 min | 29.0 min | **19.8 min** |
| Memory | 10 GB | 7.9 GB | |
| Second-look verdicts | 7 right, 1 hedged, 3 wrong | 13 right, 0 wrong | 3 right, 4 wrong, 1 unproven, 1 cannot tell |

The clean-commit alarm is the same low console-print check in all three. Qwen 3.5's second look agreed with every
claim; on this project every claim was right, so the test of whether it lets false alarms through is the other two
projects, which have more clean commits.

What the first runs looked like before the wording was fixed: 17, 16 and 15. Qwen 3.5 had named five real bugs with
the rule each breaks and answered "refuted" in the second look while its own reason agreed with the claim; the
words "against" and "refuted" were being read the other way. The fields are now `clears` and `proves`, the verdicts
`reviewer_is_right`, `reviewer_is_wrong` and `cannot_tell`.

### After more fixes, with the 14b and Qwen 3.5 (2026-11-15)

The second look was changed again after the table above: a model that agreed with every yes-or-no question
(Qwen 3.5 confirmed a claim and then its opposite 14 times of 15) is now made to choose between the claim and the
opposite, twice, in either order; the reply states what the rules expect before any finding; double quotes are
never typed in a reply. Clean runs with those changes:

| | `qwen2.5-coder:14b` | `qwen3.5:9b` |
|---|---|---|
| maze-lab, bugs found | 17 of 19 | 18 of 19 (rescored from cached replies) |
| maze-lab, clean commits with an alarm | 1 of 3 (console print) | 1 of 3 (the same) |
| records-lab, bugs found | 18 of 19 (older build) | 17 of 19 |
| records-lab, clean commits with a model alarm | 0 of 5 (older build) | **2 of 5**, both high |
| records-lab, model time | | 23.1 min |

`qwen3.5:9b` is the default from v0.7.0. The two records-lab false alarms came from the choice being made for a
reason unrelated to the claim, and from the judge not seeing a method the claim named; v0.7.0 counts a choice only
when its reasons speak of the claim, and shows the judge the types the claim names first. Those two changes are not
yet measured on the labs. To keep the 14b, set `model = { name = "qwen2.5-coder:14b" }` or `CODIST_MODEL`.
