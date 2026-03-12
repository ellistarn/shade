package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ellistarn/muse/internal/llm"
	"github.com/ellistarn/muse/internal/source"
	"github.com/ellistarn/muse/internal/storage"
)

func TestDreamCmd_RequiresBucket(t *testing.T) {
	t.Setenv("MUSE_BUCKET", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"dream"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when bucket is not set")
	}
	if got := err.Error(); got != "bucket is required: use --bucket or set MUSE_BUCKET" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestDreamCmd_LearnRequiresBucket(t *testing.T) {
	t.Setenv("MUSE_BUCKET", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"dream", "--learn"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when bucket is not set with --learn flag")
	}
}

func TestRunDream_PropagatesRunError(t *testing.T) {
	store := &failingStore{err: fmt.Errorf("storage unavailable")}
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	err := runDream(ctx, &stdout, &stderr, store, &testLLM{}, &testLLM{}, false, false, 100)
	if err == nil {
		t.Fatal("expected error from failing store, got nil")
	}
	if !strings.Contains(err.Error(), "storage unavailable") {
		t.Errorf("expected error to contain 'storage unavailable', got: %s", err.Error())
	}
}

func TestRunDream_PropagatesLearnError(t *testing.T) {
	store := newTestStore()
	store.reflections["memories/test/sess-1.json"] = "- observation"
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	err := runDream(ctx, &stdout, &stderr, store, nil, &testLLM{err: fmt.Errorf("learn failed")}, true, false, 0)
	if err == nil {
		t.Fatal("expected error from failing LLM, got nil")
	}
	if !strings.Contains(err.Error(), "learn failed") {
		t.Errorf("expected error to contain 'learn failed', got: %s", err.Error())
	}
}

func TestRunDream_SuccessfulRun(t *testing.T) {
	store := newTestStore()
	store.addSession("test", "sess-1", []source.Message{
		{Role: "user", Content: "use tabs"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "also no emojis"},
		{Role: "assistant", Content: "sure"},
	})
	mockLLM := &testLLM{
		reflectResponse: "- Uses tabs\n- No emojis",
		learnResponse: `=== SKILL: style ===
---
name: Style
description: Code style preferences.
---

Use tabs. No emojis.`,
	}

	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	err := runDream(ctx, &stdout, &stderr, store, mockLLM, mockLLM, false, false, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Processed 1 memories") {
		t.Errorf("expected 'Processed 1 memories', got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Produced 1 skills") {
		t.Errorf("expected 'Produced 1 skills', got: %s", stdout.String())
	}
}

func TestRunDream_SuccessfulLearn(t *testing.T) {
	store := newTestStore()
	store.reflections["memories/test/sess-1.json"] = "- observation"
	mockLLM := &testLLM{
		learnResponse: `=== SKILL: test ===
---
name: Test
description: Test skill.
---

Content.`,
	}

	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	err := runDream(ctx, &stdout, &stderr, store, nil, mockLLM, true, false, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Produced 1 skills") {
		t.Errorf("expected 'Produced 1 skills', got: %s", stdout.String())
	}
}

// testStore implements dream.Store with in-memory state.
type testStore struct {
	sessions    []storage.SessionEntry
	data        map[string]*source.Session
	skills      map[string]string
	reflections map[string]string
}

func newTestStore() *testStore {
	return &testStore{
		data:        map[string]*source.Session{},
		skills:      map[string]string{},
		reflections: map[string]string{},
	}
}

func (s *testStore) addSession(src, id string, messages []source.Message) {
	key := fmt.Sprintf("memories/%s/%s.json", src, id)
	s.sessions = append(s.sessions, storage.SessionEntry{
		Source:       src,
		SessionID:    id,
		Key:          key,
		LastModified: time.Now(),
	})
	s.data[src+"/"+id] = &source.Session{
		Source:    src,
		SessionID: id,
		Messages:  messages,
	}
}

func (s *testStore) ListSessions(_ context.Context) ([]storage.SessionEntry, error) {
	return s.sessions, nil
}
func (s *testStore) GetSession(_ context.Context, src, id string) (*source.Session, error) {
	sess, ok := s.data[src+"/"+id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s/%s", src, id)
	}
	return sess, nil
}
func (s *testStore) ListReflections(_ context.Context) (map[string]time.Time, error) {
	result := map[string]time.Time{}
	for key := range s.reflections {
		result[key] = time.Now()
	}
	return result, nil
}
func (s *testStore) GetReflection(_ context.Context, key string) (string, error) {
	content, ok := s.reflections[key]
	if !ok {
		return "", fmt.Errorf("reflection not found: %s", key)
	}
	return content, nil
}
func (s *testStore) PutReflection(_ context.Context, key, content string) error {
	s.reflections[key] = content
	return nil
}
func (s *testStore) DeletePrefix(_ context.Context, prefix string) error {
	if prefix == "dreams/reflections/" {
		s.reflections = map[string]string{}
	}
	return nil
}
func (s *testStore) PutSkill(_ context.Context, name, content string) error {
	s.skills[name] = content
	return nil
}
func (s *testStore) SnapshotSkills(_ context.Context, _ string) error { return nil }

// failingStore implements dream.Store where all operations return an error.
type failingStore struct{ err error }

func (s *failingStore) ListSessions(_ context.Context) ([]storage.SessionEntry, error) {
	return nil, s.err
}
func (s *failingStore) GetSession(_ context.Context, _, _ string) (*source.Session, error) {
	return nil, s.err
}
func (s *failingStore) ListReflections(_ context.Context) (map[string]time.Time, error) {
	return nil, s.err
}
func (s *failingStore) GetReflection(_ context.Context, _ string) (string, error) {
	return "", s.err
}
func (s *failingStore) PutReflection(_ context.Context, _, _ string) error { return s.err }
func (s *failingStore) DeletePrefix(_ context.Context, _ string) error     { return s.err }
func (s *failingStore) PutSkill(_ context.Context, _, _ string) error      { return s.err }
func (s *failingStore) SnapshotSkills(_ context.Context, _ string) error   { return s.err }

// testLLM implements dream.LLM for command-level tests.
type testLLM struct {
	reflectResponse string
	learnResponse   string
	err             error
}

func (m *testLLM) Converse(_ context.Context, system, _ string, _ ...llm.ConverseOption) (string, llm.Usage, error) {
	if m.err != nil {
		return "", llm.Usage{}, m.err
	}
	usage := llm.Usage{InputTokens: 100, OutputTokens: 50}
	if strings.Contains(system, "distilling observations") {
		return m.learnResponse, usage, nil
	}
	return m.reflectResponse, usage, nil
}
