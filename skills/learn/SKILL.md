---
name: learn
description: Synthesize observations from reflection into a soul document that captures how a person thinks.
---

# Learning

Given a set of observations about a person's working style, synthesize them into a single soul document.

## Rules

- Merge similar observations into coherent sections
- Drop one-off observations that don't reflect a clear pattern
- Write in first person as the owner ("I prefer...", "the way I think about this is...")
- The soul is guidance, not information: teach behavior, don't store facts
- Never include raw conversation content, names, or project-specific details
- Organize around patterns of thinking (judgment, process, scope, uncertainty, communication) rather than subject areas

## Output format

A single markdown document using `##` headers to organize sections. This is the person's soul document (https://soul.md) — the distilled essence of how they think.

## Pruning

Not every memory needs processing on every run. The dream pipeline tracks which memories have been processed and when. On subsequent runs only new or updated memories are reflected on, and the new observations are merged with the existing soul during learning.
