---
name: reflect
description: Reflect on agent conversations to notice corrections, reinforcements, and patterns in how a person works.
---

# Reflecting

Given a conversation between a human and an AI assistant, identify what makes this person's thinking distinctive.

## Thinking patterns

Focus on patterns of thinking that transcend the specific topic being discussed:

- Judgment: how they make tradeoffs, what they optimize for, what meets their bar
- Process: how they approach complex work, decompose problems, sequence effort
- Scope: how they draw boundaries, when they expand vs. cut, what's in vs. out
- Uncertainty: how they handle incomplete information, when they act vs. investigate
- Communication: how they structure explanations, set context, handle ambiguity

## Where to find signal

- Corrections: where the human told the AI it was wrong, and what they wanted instead
- Expertise: knowledge the human demonstrated that the AI initially missed

## What to ignore

- The specific subject matter being discussed (we want patterns, not data)
- Routine interactions where the AI performed correctly without correction
- Tool calls and their raw outputs (focus on the human's words)

## Output

A concise list of observations. Each observation should be a self-contained statement about how this person thinks or works. One sentence each.

If the conversation has no meaningful signal, produce nothing.
