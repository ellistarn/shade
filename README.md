# Shade

A shade is a projection of how you think. It absorbs your memories from agent interactions, distills
them into skills, and serves those skills to any agent that asks.

## How it works

**Upload** pulls memories from local agent databases on your machine (OpenCode, Claude Code, Kiro,
etc.) and syncs them to storage. Your shade learns from these memories by dreaming.

**Dream** reads your uploaded memories, focusing on the feedback you give to models: where they get
things wrong, what you correct, what you reinforce. It compresses these patterns into skills that
capture your expertise. Skills are guidance, not information: they teach models how you want things
done without leaking underlying data. Dreaming is lossy by design, keeping what matters and
forgetting what doesn't.

**Listen** starts an MCP server that exposes a single tool: **ask**. An agent sends a question and
gets back guidance shaped by your skills. Each call is stateless, a one-shot interaction with no
session history or persistence.

## How the agent loop works

When an agent asks a question, the shade runs a lightweight agent loop within a single request. It
presents the LLM with a catalog of available skills (just names and one-line descriptions), then
lets the LLM decide which skills are relevant. The LLM can call a `read_skill` tool to fetch the
full content of any skill it needs. Once it has the context it wants, it answers.

This is progressive disclosure: the LLM only sees skill content that's relevant to the question
instead of everything at once. The loop is capped at one round of tool use (ask, fetch skills,
answer) to keep latency predictable. There's no multi-turn session history and no conversation
persistence. Every call starts fresh.

A full agentic loop with multi-turn reasoning may come later, but stateless one-shot is sufficient
for now.

## Usage

```
export SHADE_BUCKET=$USER-shade
export SHADE_MODEL=claude-sonnet-4-20250514

shade upload    # sync memories to storage
shade dream     # distill skills from memories
shade listen    # start the MCP server
```

## Install

```
go install github.com/ellistarn/shade/cmd/shade@latest
```

Then add your shade as an MCP server so agents can ask it questions. For local use, add a stdio
server to your agent's MCP config:

```json
{
  "mcpServers": {
    "shade": {
      "command": "shade",
      "args": ["listen"]
    }
  }
}
```

For other operations like uploading memories or inspecting skills, use the shade CLI directly.

The MCP server can also be deployed as a hosted remote server so your shade is available to agents
running anywhere.

## Storage

S3-compatible storage with the following layout:

```
skills/{name}/SKILL.md      # distilled skills (https://agentskills.io)
memories/{source}/{id}.json # human session history
dream/state.json            # tracks which memories have been dreamed about
```
