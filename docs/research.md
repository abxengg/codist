# What the research says, and what Codist took from it

Read in November 2026: papers on model code review, defect prediction, behaviour differences, documents against
code, incident analysis, local model speed and changing model weights. The numbers below are from abstracts and
summaries; read a paper before quoting it.

## Where Codist already agrees with the field

| Finding | Source | In Codist |
|---|---|---|
| False alarms, not missed bugs, are what holds review tools back: best F1 18.7% on 1,000 real pull requests | [SWR-Bench](https://arxiv.org/abs/2509.01494) | Quote checks, exact contradiction checks, the second look, notes that open no issue |
| A detector followed by a filter reaches 75% precision in production | [BitsAI-CR](https://arxiv.org/abs/2501.15134) | Review, then second look, then the filter learned from Useful/Wrong |
| A model removes 94 to 98% of static-analysis false alarms | [Tencent study](https://arxiv.org/abs/2601.18844) | Same shape, in the other direction: exact checks correct the model |
| Logistic regression on added lines matches deep defect predictors | [LApredict](https://arxiv.org/abs/2103.07068) | `codist-review/risk` |
| Static checkers alone find few real bugs (Error Prone 7.5%, SpotBugs 6%) | [QA tools for Java](https://arxiv.org/abs/2305.16812) | Checks are for exactness; the rule set is not grown for recall |

## What was added

| Change | Why | Source | Measured here |
|---|---|---|---|
| **Numbers the documents state** (`spec-number`, no model) | Models miss code that drifts from a document that still reads plausibly | [TRACE](https://arxiv.org/html/2604.03447v1) | Reports both changed-number commits of maze-lab that the model missed in two runs; no finding on records-lab, shop-lab or demo-shop (73 commits) |
| **What became of the findings** (Repository page) | Production tools judge themselves by how often flagged lines are later changed | [BitsAI-CR](https://arxiv.org/abs/2501.15134), [Atlassian](https://arxiv.org/abs/2510.05450) | Counts only; it needs weeks of use to say anything |
| **Intent of a behaviour difference** | A fix is supposed to change results; the commit message is the oracle | [Testora](https://arxiv.org/abs/2503.18597) | Right on 4 of 4 maze-lab commits with differences (two "Simplify" commits with planted bugs marked not covered, two announced changes marked covered); too few to rely on |
| **Second reading of high-risk commits** | Combining several reviews raised F1 by up to 43.7% | [SWR-Bench](https://arxiv.org/abs/2509.01494) | With every file read twice (`twice = true`), the labs went from 57 to 60 of 62 at about double the model time (2026-11-27, measured with rules later changed; [codist-eval#4](https://github.com/abxengg/codist-eval/issues/4)) |
| **Reverted commits as bug labels** | SZZ labels are 14 to 50% noisy; a revert names its commit | [ReDef](https://arxiv.org/abs/2509.09192) | Unit tested |
| **Reads first, second looks after** | A model server reuses its work only on a prompt that starts like the last one | [llama.cpp](https://github.com/ggml-org/llama.cpp/discussions/13606) | Not measured |
| **`codist training`** | A small model with an adapter trained on accepted and rejected findings beat prompting a large one (about 76% against 64% F1, one study) | [CodeRabbit feedback study](https://arxiv.org/pdf/2607.03316) | Export only |

## Checked against research before it shipped (v0.12.0)

After the lab run of 2026-11-26 (54 of 62 planted bugs, 2 of 14 clean commits alarmed), each idea for improving the
findings was checked against published work first, and two were dropped because the research pointed the other
way. Each change kept was then measured by replaying the labs from the stored review trails. Plan and results:
[abxengg/codist#19](https://github.com/abxengg/codist/issues/19).

| Idea | What the research says | Source | Outcome here |
|---|---|---|---|
| Drop the yes-or-no "is the reviewer right?" question | A model checking its own reasoning without outside feedback does not improve it and can make it worse; self-critique collapses where sound outside checks help; models agree with what a question suggests | [Huang et al., ICLR 2024](https://arxiv.org/abs/2310.01798), [Stechly et al.](https://arxiv.org/abs/2402.08115), [Sharma et al., ICLR 2024](https://arxiv.org/abs/2310.13548) | Done. It had said yes 45 times in 49, and its "could not tell" let both clean-commit alarms of the replays through |
| Ask the A/B choice only once | Swapping the order of two options changes a model's verdict; ask in both positions | [Wang et al., ACL 2024](https://aclanthology.org/2024.acl-long.511.pdf), [Zheng et al.](https://arxiv.org/abs/2306.05685) | Dropped: both orders kept |
| Keep a choice whose answers follow the order | Send position-inconsistent cases to a person; a model's letter can follow a bias its explanation does not mention; inconsistency is highest on close calls | Wang et al., [Turpin et al., NeurIPS 2023](https://arxiv.org/abs/2305.04388), [Shi et al.](https://arxiv.org/abs/2406.07791) | Done, as a medium alarm when both reasons are about the claim: 3 real bugs kept, 1 false alarm |
| Keep a filter stage at all | A detector plus a filter reaches 75% precision in production; tools whose findings people do not act on get switched off | [BitsAI-CR](https://arxiv.org/abs/2501.15134), [Sadowski et al., CACM 2018](https://cacm.acm.org/magazines/2018/4/226371-lessons-from-building-static-analysis-tools-at-google/pdf) | Kept |
| Delete the second reading | Several reviews combined raise F1 by up to 43.67%; sampling several answers improves accuracy | [SWR-Bench](https://arxiv.org/abs/2509.01494), [Wang et al., ICLR 2023](https://arxiv.org/abs/2203.11171) | Dropped; measured instead: +3 bugs at double the time |
| Exact check for calls between services | Pair a model with sound outside checkers; static recovery of Feign calls and cross-service change impact | [Stechly et al.](https://arxiv.org/abs/2402.08115), [arXiv 2404.01789](https://arxiv.org/pdf/2404.01789), [arXiv 2501.11778](https://arxiv.org/html/2501.11778v1) | Done (`unserved-call`): finds shop-lab's cross-service bug; no report on 11 clients of Orbit Commerce |
| A limit on the wait for the first word | Client libraries keep first-word, gap and whole-call timers apart; queueing lengthens the first-word wait | [BAML](https://docs.boundaryml.com/guide/baml-basics/timeouts), [LiteLLM](https://docs.litellm.ai/docs/proxy/timeout.md), [Red Hat](https://developers.redhat.com/articles/2025/08/08/ollama-vs-vllm-deep-dive-performance-benchmarking) | Done: 3 minutes, against 63 s at most over 288 calls |
| A different model as judge | Models favour their own output; but same-model review in a fresh session matched another model on code | [Panickssery et al., NeurIPS 2024](https://arxiv.org/abs/2404.13076), [cross-model review study](https://arxiv.org/pdf/2610.01471) | Not done |

Tried and undone: asking the judge to quote the line that proves the claim. It sided with the code on three real
bugs whose proof it said it could not see (57 of 62 instead of 59), and quoting a real line did not stop a false
reason. The final rules found 60 of 62 with 1 of 14 clean commits alarmed.

## Making the parts around the model do more

Added the same week, after looking at what the model missed and what it got wrong on the labs.

| Change | Why | Measured here |
|---|---|---|
| **`behaviour-change` finding** (no model) | A commit that says it only tidies and changes results has broken its own promise; 22 real refactorings showed no difference when run | See [comparison.md](comparison.md) |
| **Run results as facts in the model's prompt** | A model told what the code returned before and after does not have to guess it | Same |
| **Describe first** (`change`, `effect`, then `findings`) | Splitting the judgement into steps is what made the 14b model's intent labels right | Same |
| **Minor model findings are notes** | In every earlier lab run, none of 32 true model findings was rated low; the false alarms were | Same |
| **Second look: redundant code is not a problem** (`judge-3`) | A clean commit had been flagged "redundant" and confirmed | Same |
| **Finding filter learns from fixed issues** | A team that fixes a finding has judged it without clicking | Unit tested |
| **One git call for a commit and its diff; merges read by their hand-written part** | 3.2 git calls a commit; merges were not reviewed | 1,409 to 1,048 calls and 30.1 s to 25.1 s on 442 commits |
| **Reading plan for every changed file** | The model stopped after 8 files and said nothing about the rest; now each file is read or skipped with a reason, and a skipped one can be read on request | Unit tested; the planning call was not yet tried on a large real commit |
| **Sandbox for behaviour runs** (`sandbox-exec`, `bwrap`) | The project's code could read the home folder and reach the network | Tested on macOS |

## Ways around what a small model can find

A second reading of the research asked what to do when the model itself is the limit. In every strong result the
model does a narrow job and something exact does the rest. Built from that list:

| Idea | Source | In Codist |
|---|---|---|
| Plant bugs in the repository's own code and count what is caught | mutation scoring of reviewers | `codist-eval/mutants`: on maze-lab, running the code exposed 22 of 40 planted bugs with no model, and 12 became issues |
| Use idle time; ask specialists, not one generalist | [SWR-Bench](https://arxiv.org/abs/2509.01494), [VulTrial](https://conf.researchr.org/details/icse-2026/icse-2026-research-track/214/Let-the-Trial-Begin-A-Mock-Court-Approach-to-Vulnerability-Detection-using-LLM-Based), [MultiVer](https://arxiv.org/pdf/2604.21282) | `deep = true`: two specialist readings of recent commits when the model is idle |
| Properties checked by running | [DiffTestGen](https://arxiv.org/pdf/2607.16024) | Two exact ones without a model: a call must not start crashing; a result that was never negative must not become negative |
| Show degrees of belief | precision against recall | Findings are tagged proven, likely or to judge |

Built after: checks the model writes from fixed bugs (learned checks, proved against the fix and the rest of the
code before they are kept, as [KNighter](https://arxiv.org/abs/2503.09002) does), method promises compared before and
after a commit and joined to the callers from the call graph (as [LLMDFA](https://arxiv.org/abs/2402.10754) splits
the job), and an experiment on whether a local model finds planted lines more surprising than the originals
([naturalness of buggy code](https://arxiv.org/abs/1506.01159)). Results are in [comparison.md](comparison.md).

## What was removed

- **Callers as context.** 5 of 10 without, 4 of 10 with, and [AACR-Bench](https://arxiv.org/abs/2601.19494) found no
  context method that helps across models.
- **`/api/working` and `/api/commits/{sha}/impact`.** Both always answered empty; the Impact card went with them.
- **Nine interface components** for pages that were switched off (Ask, Branches, Branch graph, Memory, People,
  Developer profile, Release, Working plan, Impact): about 1,100 lines.

## Changing the model's weights

- **SEAL** ([Self-Adapting Language Models](https://alphaxiv.org/abs/2506.10943), MIT): the model writes its own
  finetuning data and is trained on it, with reinforcement learning rewarding the self-edits that help. Reported: 47%
  on taking in new knowledge, 72.5% on a subset of ARC. It needs many training rounds and forgets earlier knowledge
  as it goes, so it is not something a laptop tool can run after each commit.
- **Code2LoRA** ([paper](https://papers.cool/arxiv/2606.06492)): a network that writes an adapter for a repository
  without training on it. Shown on Python test-assertion completion only.
- **An adapter from accept and reject history** (above): the one result that maps onto data Codist already has.
  That is why the export exists and why training itself is left to the tools made for it.
- **Steering activations** without training ([secure code](https://arxiv.org/html/2507.09508v1),
  [false alarms across languages](https://arxiv.org/pdf/2604.27714)): fewer false alarms, but recall fell from 0.95
  to 0.72 in one condition, and Ollama gives no access to activations.

## Not solved anywhere

- No careful study of small local models for review was found.
- Going from a crash to the commit that caused it is described as open; work such as
  [AutoCrashFL](https://arxiv.org/abs/2510.22530) stops at the file.
