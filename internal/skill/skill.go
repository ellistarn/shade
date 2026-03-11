package skill

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

// Skill represents a parsed SKILL.md file following the Agent Skills spec.
type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"-"` // full markdown body after frontmatter
}

// LoadAll fetches all skills from S3 under the skills/ prefix.
// Each skill lives at skills/{name}/SKILL.md.
func LoadAll(ctx context.Context, client *s3.Client, bucket string) ([]Skill, error) {
	paths, err := listSkillPaths(ctx, client, bucket)
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, path := range paths {
		sk, err := loadSkill(ctx, client, bucket, path)
		if err != nil {
			continue // skip unparseable skills
		}
		skills = append(skills, *sk)
	}
	return skills, nil
}

func listSkillPaths(ctx context.Context, client *s3.Client, bucket string) ([]string, error) {
	prefix := "skills/"
	var paths []string
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list skills: %w", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if strings.HasSuffix(key, "/SKILL.md") {
				paths = append(paths, key)
			}
		}
	}
	return paths, nil
}

func loadSkill(ctx context.Context, client *s3.Client, bucket, key string) (*Skill, error) {
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", key, err)
	}
	return parse(string(data))
}

// parse splits a SKILL.md into YAML frontmatter and markdown body.
func parse(raw string) (*Skill, error) {
	frontmatter, body, err := splitFrontmatter(raw)
	if err != nil {
		return nil, err
	}
	var sk Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &sk); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	sk.Content = strings.TrimSpace(body)
	return &sk, nil
}

// splitFrontmatter extracts YAML between --- delimiters and the remaining body.
func splitFrontmatter(raw string) (frontmatter, body string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	// find opening ---
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "---" {
			break
		}
	}
	// collect frontmatter until closing ---
	var fm strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		fm.WriteString(line)
		fm.WriteString("\n")
	}
	if fm.Len() == 0 {
		return "", "", fmt.Errorf("no frontmatter found")
	}
	// remainder is body
	var b strings.Builder
	for scanner.Scan() {
		b.WriteString(scanner.Text())
		b.WriteString("\n")
	}
	return fm.String(), b.String(), nil
}
