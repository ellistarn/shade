# Incremental Distillation

## Problem

Batch distillation reprocesses all observations every run. A 200k context window holds ~100-130
conversations before the learn step overflows. Clustered distillation solves this but introduces six
pipeline stages, an embedding model dependency, a custom HDBSCAN implementation, and a local-only
artifact store. We need to update a document with new information without rereading everything.

## Insight

The question isn't "what does the complete set of observations say?" — it's "given who we think
this person is, what does this new evidence change?"

```
muse(n+1) = update(muse(n), new_observations)
```

Each update folds new observations into the existing muse. The muse is sticky — things persist
unless the new evidence gives reason to change them. This is editing a document, not updating a
moving average.

## Design

### Pipeline

```
conversations ─► OBSERVE ─► observations ─► UPDATE ─► (muse', forgotten)

                ┌───────────────────────────────────────────────────────┐
 OBSERVE        │  Per-conversation LLM call (parallel, Sonnet)        │
                │  Same as today — extract + refine per conversation    │
                │  → zero or more observations per conversation         │
                └───────────────────────────────────────────────────────┘

                ┌───────────────────────────────────────────────────────┐
 UPDATE         │  Single LLM call (Opus, extended thinking)           │
                │  Input: current muse + new observations              │
                │  Output: updated muse + what was forgotten           │
                └───────────────────────────────────────────────────────┘
```

Two stages, two model tiers. The context window holds only the current muse plus one batch of
observations, never the full history.

### Update granularity

The natural batch is a sync: a few new conversations producing ~10-15 observations. One observation
at a time works but costs an Opus call per observation. Batching amortizes cost.

### How the update works

The muse is ground truth. The update only changes what new evidence gives reason to change:

- **Add** new patterns not yet in the muse.
- **Strengthen** patterns the new evidence reinforces (higher confidence, more specificity).
- **Weaken or remove** patterns the new evidence contradicts.
- **Leave everything else alone.** Absence of evidence is not evidence of absence. A pattern
  observed once and never contradicted persists indefinitely.

Forgetting comes from contradiction or subsumption, not from time passing. "I used to prefer tabs,
now I use spaces" is a reason to forget. Six months of silence about tabs is not.

### The forgotten log

Each update produces two outputs:

1. **muse.md** — the updated muse
2. **forgotten.md** — what was removed or softened, with a reason

Every entry in the forgotten log has a cause: "contradicted by X" or "subsumed by Y." If the log
ever says "hasn't been mentioned recently," the update prompt is wrong.

The forgotten log provides:

- **Audit.** Why did the muse stop mentioning X?
- **Recovery.** Feed a dropped observation back in to restore it.

### Storage

Uses the existing versioning structure with one new file:

```
~/.muse/
├── conversations/{source}/{session_id}.json
├── observations/{source}/{session_id}.md
├── muse/versions/{timestamp}/
│   ├── muse.md
│   ├── diff.md
│   └── forgotten.md                                  # new
```

### The update prompt

- The muse is ground truth. Only change what new evidence gives reason to change.
- New patterns are added tentatively. Confidence grows with repeated observation.
- Contradicted patterns weaken or are removed. The forgotten log records what and why.
- Reinforced patterns strengthen.
- Everything else stays.

### Order independence

There is no canonical ordering of observations. They arrive from multiple machines and sources.
Framing the update as "reinforce, add, or weaken relative to the current muse" is naturally
order-insensitive — the muse is the state, observations are perturbations.

## Commands

```bash
muse distill                     # observe new conversations, update muse
muse distill --learn             # re-update from recent observations without re-observing
muse distill --reobserve         # re-observe from scratch, then update
muse distill kiro                # only observe kiro conversations, then update
```

`muse show --diff` and `muse show --forgotten` display the respective logs.

## Bootstrap

First run (no existing muse): select the ~200 most recent observations and run a single learn
call to produce the initial muse. Older observations are not lost — they can be folded in through
subsequent incremental updates if they keep surfacing.

`muse distill --rebuild` re-bootstraps from recent observations. This is disaster recovery.

## Cost

The observe step is shared across all approaches. Post-observe cost per sync (3 new conversations,
~15 new observations, ~15 clusters, Opus with 16k thinking budget):

| | Map-reduce | Clustered | Incremental |
|---|---|---|---|
| Classify | — | ~$0.07 | — |
| Embed | — | ~$0.01 | — |
| Synthesize | — | ~$0.90 | — |
| Learn/Merge/Update | ~$5+ (300k in) | ~$1.80 (20k in) | ~$1.60 (7k in) |
| **Total** | **~$5+** | **~$2.80** | **~$1.60** |

Thinking tokens dominate the Opus call in both clustered and incremental. The Sonnet calls in
clustered add ~$1 per sync. Incremental is ~40% cheaper than clustered per sync — meaningful
but not dramatic.

The cost advantage widens at scale: map-reduce's Opus input grows linearly with total observations
and eventually overflows. Clustered's synthesize step grows with cluster count. Incremental's
update is always ~7k tokens — the muse plus the new batch.

The primary advantage is simplicity, not cost.

## Decisions

### Why incremental over clustered?

Both solve context overflow. Clustering groups and summarizes before merging — six stages, four
API calls, an embedding model, a custom clustering algorithm. Incremental updates never look at all
observations at once — two stages, one API call.

Clustering gives thematic organization within each run. Incremental updates rely on the muse
accumulating structure over time. The clustered merge prompt says "don't preserve cluster boundaries
if a better structure emerges" — the thematic organization is already expected to be reorganized.

### Why sticky over decaying?

Important things are often infrequent. Annual review feedback, career decisions, core values —
these appear rarely in observations but matter persistently. A decay model (EWMA) conflates
frequency with importance: anything not continuously reinforced fades. This means re-learning
the same things every cycle.

A sticky model preserves everything unless contradicted. The muse doesn't forget your annual
review feedback just because nobody mentioned it for six months. It forgets it when new evidence
says something different.

## Deferred

### Configurable bootstrap window

Bootstrap uses the ~200 most recent observations. Users may want to control this — larger windows
for comprehensive initial muses, smaller for faster starts. **Revisit when:** users ask for it.

### Conflict-aware sync

Concurrent updates from different machines: last write wins. The forgotten log and `--rebuild`
provide recovery. **Revisit when:** concurrent updates cause real problems.
