# Incremental Distillation

## Problem

Reprocessing all observations every update is wasteful and degrades quality. As context grows,
LLMs produce lower-quality responses (context rot), and eventually the input overflows entirely.

[Exponentially weighted moving averages](https://en.wikipedia.org/wiki/EWMA) solve a similar
problem in statistics: maintain a running estimate from streaming data without revisiting history.
The current estimate plus the new data point is sufficient. In EWMA, recent data points have
higher weight, which naturally bounds the estimate's length by forgetting stale signal. The same
intuition applies here: the current muse plus new observations is sufficient to produce the next
muse.

## Solution

```
muse(n+1) = update(muse(n), new_observations)
```

Each update folds new observations into the existing muse. The full history has influence
through the muse itself: prior observations shaped it, and some were reinforced while others
were not. The update input is the current muse plus one batch of observations.

There are two length problems. **Distillation length**: more observations than fit in context.
**Muse length**: as the muse grows, it becomes less effective as a system prompt. This design
solves distillation length. Muse length is a follow-up.

```
conversations ─► OBSERVE ─► observations ─► UPDATE ─► muse'
```

**Observe** extracts and refines observations per conversation. Parallel, Sonnet-tier. Shared
across all distillation methods.

**Update** is a single Opus call with extended thinking. Input: current muse + new batch.
Output: updated muse.

### Update granularity

Each run of `muse distill` produces a batch: a few new conversations yielding ~10-15 observations.
Updating one observation at a time works but costs an Opus call per observation. Batching amortizes
that overhead, but too large a batch reintroduces context rot. The right batch size balances call
overhead against context quality and needs experimental tuning.

### How the update works

The update is conservative. It starts from the muse as-is and only modifies what the new
observations give specific reason to modify. There is no numeric confidence score. The weight
of evidence is conveyed through natural language:

- **Add** new patterns not yet in the muse using hedged language ("may prefer short functions").
  As more observations confirm the pattern, the language becomes more direct ("consistently
  prefers short functions").
- **Strengthen** existing patterns by making language more direct or adding specificity. "Tends
  to prefer X" becomes "prefers X, especially in context Y."
- **Weaken or remove** patterns the new evidence contradicts. "Prefers tabs" becomes "has used
  both tabs and spaces" or is removed entirely.
- **Leave everything else alone.** Absence of evidence is not evidence of absence. A pattern
  observed once and never contradicted persists indefinitely.

### Storage

Observations are shared across distillation methods.

```
~/.muse/
├── conversations/{source}/{session_id}.json              # input, syncable
├── distill/
│   └── observations/{source}/{session_id}.json           # shared, syncable
├── muse/versions/{timestamp}/
│   ├── muse.md                                           # output, syncable
│   └── diff.md                                           # output, syncable
```

### Strategies

```bash
muse distill --method=incremental
```

## Bootstrap

First run (no existing muse): select the ~200 most recent observations and run a single update
call to produce the initial muse.

## Decisions

### Why ~10 observations per batch?

Each Opus call has fixed overhead from the thinking budget. Batching amortizes that. But context
rot degrades output quality as the batch grows. We start with ~10 (roughly one sync's worth of
new conversations) as a sensible default and tune experimentally.

### Why bias toward the existing muse?

The update treats the existing muse as the starting point and requires new observations to justify
changes. One observation can't rewrite the muse. This is analogous to a low learning rate in EWMA:
new data adjusts the estimate gradually. The alternative (high trust in new data) risks instability
where a single unusual conversation reshapes the muse. The tradeoff is that genuine changes in how
the owner thinks take several observations to fully propagate.

### How does observation strength work?

The strength of an observation comes from the user's input. A correction ("no, do it this way")
is stronger than a passing preference. The observe step captures this in how it phrases the
observation, and the update LLM infers relative strength from that language. We don't add explicit
weighting beyond what the user's own words convey.

## Deferred

### Muse length

As observations accumulate, the muse grows. A long muse is less effective as a system prompt
(context rot applies to the consumer too). This design solves distillation length. Muse length
requires separate mechanisms (summarization, tiered storage, or selective loading).

### Batch size tuning

The ~10 observation default is a starting point. Optimal batch size depends on observation density,
muse complexity, and model capability. Requires experimental evaluation.
