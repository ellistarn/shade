package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ellistarn/muse/internal/awsconfig"
	"github.com/ellistarn/muse/internal/source"
)

const soulKey = "soul.md"

type Client struct {
	s3     *s3.Client
	bucket string
}

// S3 returns the underlying S3 client for direct use.
func (c *Client) S3() *s3.Client { return c.s3 }

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string { return c.bucket }

func NewClient(ctx context.Context, bucket string) (*Client, error) {
	cfg, err := awsconfig.Load(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{
		s3:     s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// SessionEntry is the metadata returned by ListSessions without downloading full content.
type SessionEntry struct {
	Source       string
	SessionID    string
	Key          string
	LastModified time.Time
}

// ListSessions returns all session keys with their S3 LastModified timestamps.
func (c *Client) ListSessions(ctx context.Context) ([]SessionEntry, error) {
	var entries []SessionEntry
	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: aws.String("memories/"),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			src, id := parseSessionKey(key)
			if src == "" {
				continue
			}
			entries = append(entries, SessionEntry{
				Source:       src,
				SessionID:    id,
				Key:          key,
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
	}
	return entries, nil
}

// PutSession uploads a session as JSON and returns the number of bytes written.
func (c *Client) PutSession(ctx context.Context, session *source.Session) (int, error) {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal session: %w", err)
	}
	key := sessionKey(session.Source, session.SessionID)
	contentType := "application/json"
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to upload session %s: %w", session.SessionID, err)
	}
	return len(data), nil
}

// GetSession downloads and deserializes a session from S3.
func (c *Client) GetSession(ctx context.Context, src, sessionID string) (*source.Session, error) {
	key := sessionKey(src, sessionID)
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read session %s: %w", sessionID, err)
	}
	var session source.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session %s: %w", sessionID, err)
	}
	return &session, nil
}

// GetSoul downloads the soul document from S3.
func (c *Client) GetSoul(ctx context.Context) (string, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    aws.String(soulKey),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get soul: %w", err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read soul: %w", err)
	}
	return string(data), nil
}

// PutSoul writes the soul document to S3.
func (c *Client) PutSoul(ctx context.Context, content string) error {
	contentType := "text/markdown"
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         aws.String(soulKey),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put soul: %w", err)
	}
	return nil
}

// SnapshotSoul copies the current soul to dreams/history/{timestamp}/soul.md.
func (c *Client) SnapshotSoul(ctx context.Context, timestamp string) error {
	dstKey := fmt.Sprintf("dreams/history/%s/soul.md", timestamp)
	copySource := fmt.Sprintf("%s/%s", c.bucket, soulKey)
	_, err := c.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &c.bucket,
		CopySource: &copySource,
		Key:        &dstKey,
	})
	if err != nil {
		return fmt.Errorf("failed to snapshot soul: %w", err)
	}
	return nil
}

// GetJSON downloads and unmarshals a JSON object from S3.
func (c *Client) GetJSON(ctx context.Context, key string, v any) error {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to get %s: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", key, err)
	}
	return json.Unmarshal(data, v)
}

// PutJSON marshals and uploads a JSON object to S3.
func (c *Client) PutJSON(ctx context.Context, key string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", key, err)
	}
	contentType := "application/json"
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put %s: %w", key, err)
	}
	return nil
}

// PutReflection writes a reflection to S3 under dreams/reflections/{key}.md.
func (c *Client) PutReflection(ctx context.Context, key, content string) error {
	// Replace the memories/ prefix so reflections mirror the memory layout
	path := fmt.Sprintf("dreams/reflections/%s.md", strings.TrimPrefix(strings.TrimSuffix(key, ".json"), "memories/"))
	contentType := "text/markdown"
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &path,
		Body:        bytes.NewReader([]byte(content)),
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put reflection for %s: %w", key, err)
	}
	return nil
}

// ListReflections returns the keys of all persisted reflections under dreams/reflections/.
func (c *Client) ListReflections(ctx context.Context) (map[string]time.Time, error) {
	reflections := map[string]time.Time{}
	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: aws.String("dreams/reflections/"),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list reflections: %w", err)
		}
		for _, obj := range page.Contents {
			// Convert dreams/reflections/opencode/ses_abc.md back to memories/opencode/ses_abc.json
			key := aws.ToString(obj.Key)
			memoryKey := strings.TrimPrefix(key, "dreams/reflections/")
			memoryKey = "memories/" + strings.TrimSuffix(memoryKey, ".md") + ".json"
			reflections[memoryKey] = aws.ToTime(obj.LastModified)
		}
	}
	return reflections, nil
}

// GetReflection downloads a reflection's content from S3.
func (c *Client) GetReflection(ctx context.Context, memoryKey string) (string, error) {
	path := fmt.Sprintf("dreams/reflections/%s.md", strings.TrimPrefix(strings.TrimSuffix(memoryKey, ".json"), "memories/"))
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get reflection for %s: %w", memoryKey, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read reflection for %s: %w", memoryKey, err)
	}
	return string(data), nil
}

// DeletePrefix removes all objects under a given S3 prefix.
func (c *Client) DeletePrefix(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: &c.bucket,
				Key:    &key,
			}); err != nil {
				return fmt.Errorf("failed to delete %s: %w", key, err)
			}
		}
	}
	return nil
}

// ListDreams returns timestamps of all dream snapshots, sorted ascending.
func (c *Client) ListDreams(ctx context.Context) ([]string, error) {
	prefix := "dreams/history/"
	out, err := c.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    &c.bucket,
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list dreams: %w", err)
	}
	var timestamps []string
	for _, cp := range out.CommonPrefixes {
		p := aws.ToString(cp.Prefix)
		// "dreams/history/2026-03-11T16:30:00Z/" -> "2026-03-11T16:30:00Z"
		p = strings.TrimPrefix(p, prefix)
		p = strings.TrimSuffix(p, "/")
		if p != "" {
			timestamps = append(timestamps, p)
		}
	}
	sort.Strings(timestamps)
	return timestamps, nil
}

// GetDreamSoul loads the soul from a specific dream snapshot.
func (c *Client) GetDreamSoul(ctx context.Context, timestamp string) (string, error) {
	key := fmt.Sprintf("dreams/history/%s/soul.md", timestamp)
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get dream soul for %s: %w", timestamp, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read dream soul for %s: %w", timestamp, err)
	}
	return string(data), nil
}

func sessionKey(src, sessionID string) string {
	return fmt.Sprintf("memories/%s/%s.json", src, sessionID)
}

// parseSessionKey extracts source and session ID from a key like "memories/opencode/ses_abc.json".
func parseSessionKey(key string) (src, sessionID string) {
	// key format: memories/{source}/{session_id}.json
	key = strings.TrimPrefix(key, "memories/")
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	src = parts[0]
	sessionID = strings.TrimSuffix(parts[1], ".json")
	if src == "" || sessionID == "" {
		return "", ""
	}
	return src, sessionID
}
