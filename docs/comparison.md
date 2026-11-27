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

## v0.8.0 against v0.7.0 (2026-11-15 and 16, `qwen3.5:9b`, clean runs)

| Project | Bugs found, v0.7.0 | Bugs found, v0.8.0 build | Clean commits with a model alarm, v0.8.0 build | Model time per commit |
|---|---|---|---|---|
| records-lab | 17 of 19 | 18 of 19 | 1 of 5 | 52 s, then 91 s |
| shop-lab | 20 of 24 | 21 of 24 | 0 of 6 | 75 s, then 82 s |
| maze-lab | 18 of 19 | 18 of 19 | 0 of 3 | 100 s, then 102 s |
| all | 55 of 62 | 57 of 62 | 1 of 14 (v0.7.0: 1 of 14) | |

The two extra bugs are within the difference between two runs of one version: on maze-lab the total stayed at 18 while
the missed bug changed. The v0.8.0 build that was measured asked about method promises in every review: six findings
(five notes, one alarm), none on a planted bug, for 24 to 70% more model calls. So in the released v0.8.0 those
questions are asked only in the deep pass.

**Learned checks** (`go run ./mutants -learn` in codist-eval): a check learned for 5 of 15 kinds of planted bug; those
checks caught 1 of 42 other bugs of their kinds and matched 0 lines of clean code.

**A second reading** (`go run ./review` in codist-eval, 10 real bugs): 3 of 10 located with one reading, 3 of 10 with two.

**Line surprise** (`surprise/surprise.py` in codist-eval, 40 planted bugs on maze-lab): the planted line was more
surprising to the model than the line it replaced in 37 of 40, by 0.38 nats a piece on average. It compares a line
with its own changed twin, so it supports surprise as a ranking of where to read, not as a finding.

**The deep pass** (`CODIST_DEEP=1`, 90 idle minutes on maze-lab after its review, 106 model calls): every one of the 25
commits was read again through both lenses, about two minutes a commit and 26 for the 14-file first commit. It added
one alarm, true, on a bug the spec-number check had already raised ("Javadoc claims 1/60s but code uses 1/30s,
violating spec rule 2"), and two notes that look wrong. No new bug was found (18 of 19 before and after) and no clean
commit got an alarm. On this project it did not pay for its time; it stays off unless turned on.

## Real bugs, v0.9.0 (2026-11-18)

`go run ./review -once` in [codist-eval](https://github.com/abxengg/codist-eval/tree/main/review): 120 bugs from
Defects4J and GitBug-Java (the development half of the old benchmark). For each, the fix is reversed in the fixed
version and committed with a neutral message ("Update <files>"), then reviewed once by codist-review v0.13.0 with
`qwen3.5:9b`, each from an empty store. One bug's repository is gone from GitHub, so 119 count. 5.2 hours, about 2.6
minutes a bug.

| | Bugs of 119 |
|---|---|
| A finding within 3 lines of the bug | 67 (64 from the model) |
| **Named the bug** | **49** |
| Related point near it | 12 |
| Right line, wrong reason | 6 |
| Named, shown as an alarm (not a note) | 28 |

Against `qwen2.5-coder:14b` on the same benchmark (2026-11-04, before the claim checks): it named 51 of 118, with 31
"right line, wrong reason" hits against 6 now, in about 2.5 times the time. **The two models name different bugs**:
25 both, 24 only the 9b, 26 only the 14b, 75 together. A second, different reading is where most is left
([codist-review#8](https://github.com/abxengg/codist-review/issues/8)).

What this does not show: false alarms (the fixed versions were not reviewed), and the labels are one reader's
comparison of each finding with the real fix. A first attempt committed each bug as "Reintroduce the bug", which the
model read; those results were thrown away. Labels and results:
[codist-eval/review/results](https://github.com/abxengg/codist-eval/tree/main/review/results).


## v0.11.2 against v0.12.0 (2026-11-26 and 27)

v0.11.2 was run on fresh copies of the three labs, built by their scripts, with `qwen3.5:9b` reading every commit.
v0.12.0 was measured by replaying the same labs from that run's cached first readings with the new rules, so the
difference is the rules, not the model's mood; files whose first reading had failed were read afresh.

| Project | Bugs found, v0.11.2 | v0.12.0 | Clean commits with an alarm, v0.11.2 | v0.12.0 |
|---|---|---|---|---|
| shop-lab | 20 of 24 | 23 of 24 | 0 of 6 | 0 of 6 |
| records-lab | 18 of 19 | 18 of 19 | 1 of 5 | 0 of 5 |
| maze-lab | 16 of 19 | 19 of 19 | 1 of 3 | 1 of 3 |
| all | 54 of 62 | **60 of 62** | 2 of 14 | **1 of 14** |

- **5 of v0.11.2's 8 misses had been found by the model** and hidden by Codist's filters: three choices whose
  letters followed the order while both reasons argued the bug, one quote of the removed line, and one null claim
  the change backed by removing a `getOrDefault`. Kept now.
- **One from the new `unserved-call` check**: shop-lab's Feign client calling `/payments/charges`, which
  payment-service does not serve.
- **The records-lab alarm is gone**: the yes-or-no second look's "could not tell" (and, in the first run, a failed
  call) had kept the claim without the choice. Now every serious claim goes to the choice, which sided with the code.
- **The maze-lab alarm changed**: v0.11.2's was the console-print check on an error message in a project with no
  logger (now not flagged); v0.12.0's is a model claim on "Add M to mute" whose two answers split, kept at medium.
- **Tried and undone**: making the judge quote proof gave 57 of 62.
- **Fix compile rate** (v0.11.2, shop-lab, 10 open issues): 5 compiled on the first try, 9 after retrying with the
  compiler's errors.

Plan, research and every replay: [abxengg/codist#19](https://github.com/abxengg/codist/issues/19).
