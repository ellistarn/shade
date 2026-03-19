# Muse

An AI that thinks like you. Derived from your conversation history
across Claude Code, Kiro, and OpenCode.

## Install

```
go install github.com/ellistarn/muse@latest
```

## Getting Started

```bash
muse distill              # discover conversations and distill muse.md
muse ask "your question"  # ask your muse directly
muse listen               # start MCP server
muse show                 # print muse.md
```

After distilling, your muse is always available at `~/.muse/muse.md`. Point your
agent's system prompt to source it automatically:

```json
// ~/.config/opencode/opencode.json
{
  "instructions": ["~/.muse/muse.md"]
}
```

Or wire up the MCP server so agents can ask your muse questions on demand:

```json
{
  "mcpServers": {
    "${USER}": {
      "command": "muse",
      "args": ["listen"]
    }
  }
}
```

## Sources

Conversations are automatically discovered from:

- **Claude Code** — `~/.claude/projects/`
- **Kiro** — `~/Library/Application Support/Kiro/User/globalStorage/kiro.kiroagent/workspace-sessions/`
- **OpenCode** — `~/.local/share/opencode/opencode.db`
- **Codex** — `~/.codex/`

## Storage

By default, data is stored locally at `~/.muse/`. To use an S3 bucket instead
(for sharing across machines or hosted deployment), set the `MUSE_BUCKET`
environment variable:

```bash
export MUSE_BUCKET=$USER-muse
```

```bash
muse sync local s3                  # push everything to S3
muse sync s3 local conversations    # pull conversations from S3
```

Run `muse --help` for detailed usage.
