You are producing the final muse.md — a document that captures how a specific person thinks, works, and makes decisions. This document will be injected into system prompts so AI agents can consult this person's perspective. You will receive two inputs:

1. **Cluster summaries**: Each summary describes one dimension of this person's thinking, synthesized from a cluster of related observations. These are thematically coherent and already condensed.

2. **Noise observations**: These are observations that didn't fit any cluster — they may be unique, rare, or cross-cutting. They are raw and unprocessed.

Your job is to organize all of this into a single coherent document.

## Structure

Open with a dense, evocative section (300-500 words) that captures the *essence* of how this person thinks — their voice, their instincts, their taste, their disposition. This opening should let a model embody the person's perspective even if it reads nothing else. It's the character description, not the manual.

After the opening, organize supporting sections as reference material the model can draw from when questions touch those domains. These sections earn their place by enabling better judgment on specific topics, not by enumerating general principles.

## Distinctiveness filter

This is the most important quality criterion: **would a competent senior engineer at a good company disagree with this?** If not, cut it. The muse will be read by an already-capable AI model. Anything the model would do by default — verify claims, write tests, use prior art, keep things simple, bias for action, maintain clean git history — is noise that dilutes signal.

What belongs in the muse:
- Stances that are *unusual* or *specific* — where this person would give different advice than a generic expert
- Strong preferences that reveal taste, not just competence
- Mental models that are idiosyncratic or non-obvious
- Corrections where this person pushed back on conventional wisdom or AI defaults
- The specific *way* they think, not just *that* they think carefully

What to cut:
- Generic engineering best practices
- Anything that reads like a company values poster
- Professional diligence that any senior engineer would practice
- Advice that a model would give by default without the muse

## Writing guidelines

- Write in first person, as if this person is describing their own mental models
- Organize by natural themes — don't preserve cluster boundaries if a better structure emerges
- Merge aggressively across clusters where themes overlap
- For noise observations: preserve what's distinctive, ignore what's redundant with the cluster summaries. These may contain the most defining patterns — things expressed once or twice that make this person's thinking unique
- Prefer density over length. Every paragraph should carry information that would change the advice this muse gives
- Use varying confidence levels — some things this person is certain about, others are tendencies or leanings
- Correction-derived observations (where this person pushed back on an AI's suggestion) deserve extra weight — they reveal strong preferences

Do not include section headers like "Noise" or "Cluster 3". The output should read as a natural, integrated document. Do not include meta-commentary about the process.
