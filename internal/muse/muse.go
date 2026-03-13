package muse

import (
	"context"
	"fmt"

	"github.com/ellistarn/muse/internal/bedrock"
	"github.com/ellistarn/muse/internal/log"
	"github.com/ellistarn/muse/internal/source"
	"github.com/ellistarn/muse/internal/storage"
	"github.com/ellistarn/muse/prompts"
)

// UploadResult summarizes what happened during an upload sync.
type UploadResult struct {
	Total    int
	Uploaded int
	Skipped  int
	Bytes    int
	Warnings []string
}

// Muse holds the state needed for all operations.
type Muse struct {
	storage *storage.Client
	bedrock *bedrock.Client
	soul    string // the full soul document, loaded at init
}

func New(ctx context.Context, bucket string) (*Muse, error) {
	storageClient, err := storage.NewClient(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	bedrockClient, err := bedrock.NewClient(ctx, bedrock.ModelOpus)
	if err != nil {
		return nil, fmt.Errorf("failed to create bedrock client: %w", err)
	}
	soul, err := storageClient.GetSoul(ctx)
	if err != nil {
		if !storage.IsNotFound(err) {
			return nil, fmt.Errorf("failed to load soul: %w", err)
		}
		soul = "" // no soul yet — first run before any dreams
	}
	if soul != "" {
		log.Printf("Loaded soul (%d bytes)\n", len(soul))
	} else {
		log.Println("No soul found (run 'muse dream' to generate one)")
	}
	return &Muse{
		storage: storageClient,
		bedrock: bedrockClient,
		soul:    soul,
	}, nil
}

// NewForTest creates a Muse with caller-provided dependencies.
func NewForTest(bedrockClient *bedrock.Client, soul string) *Muse {
	return &Muse{
		bedrock: bedrockClient,
		soul:    soul,
	}
}

var systemPrompt = prompts.Muse

// Ask answers a question using the muse's soul document.
// The soul is included directly in the system prompt — no tool calling,
// no progressive disclosure. Single-shot stateless interaction.
func (m *Muse) Ask(ctx context.Context, question string) (string, error) {
	soul := m.soul
	if soul == "" {
		soul = "No soul document available yet. Run 'muse dream' to generate one from memories."
	}
	system := fmt.Sprintf(systemPrompt, soul)
	answer, _, err := m.bedrock.Converse(ctx, system, question)
	return answer, err
}

// Upload scans local sources, diffs against S3, and uploads changed sessions.
func (m *Muse) Upload(ctx context.Context) (*UploadResult, error) {
	log.Println("Listing remote sessions...")
	existing, err := m.storage.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote sessions: %w", err)
	}
	log.Printf("Found %d remote sessions\n", len(existing))
	remote := map[string]storage.SessionEntry{}
	for _, e := range existing {
		remote[e.Key] = e
	}

	log.Println("Scanning local sessions...")
	var local []source.Session
	var warnings []string
	if sessions, err := source.OpenCodeSessions(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to read OpenCode sessions: %v", err))
	} else {
		log.Printf("Found %d OpenCode sessions\n", len(sessions))
		local = append(local, sessions...)
	}
	if sessions, err := source.ClaudeCodeSessions(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to read Claude Code sessions: %v", err))
	} else {
		log.Printf("Found %d Claude Code sessions\n", len(sessions))
		local = append(local, sessions...)
	}
	if sessions, err := source.KiroSessions(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to read Kiro sessions: %v", err))
	} else {
		log.Printf("Found %d Kiro sessions\n", len(sessions))
		local = append(local, sessions...)
	}

	log.Printf("Diffing %d local sessions against remote...\n", len(local))
	var uploaded, skipped int
	var totalBytes int
	for i := range local {
		sess := &local[i]
		key := fmt.Sprintf("memories/%s/%s.json", sess.Source, sess.SessionID)
		if entry, exists := remote[key]; exists {
			if !sess.UpdatedAt.After(entry.LastModified) {
				log.Printf("  skip (unchanged) %s\n", key)
				skipped++
				continue
			}
		}
		n, err := m.storage.PutSession(ctx, sess)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to upload %s: %v", sess.SessionID, err))
			continue
		}
		log.Printf("  upload (%s) %s\n", FormatBytes(n), key)
		uploaded++
		totalBytes += n
	}
	return &UploadResult{
		Total:    len(local),
		Uploaded: uploaded,
		Skipped:  skipped,
		Bytes:    totalBytes,
		Warnings: warnings,
	}, nil
}

func FormatBytes(b int) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// Soul returns the current soul document for use by the MCP handler.
func (m *Muse) Soul() string {
	return m.soul
}
