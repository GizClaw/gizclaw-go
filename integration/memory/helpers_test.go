package memoryintegration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/haivivi/giztoy/go/pkg/kv"
	"github.com/haivivi/giztoy/go/pkg/memory"
	"github.com/haivivi/giztoy/go/pkg/recall"
)

const integrationSeparator byte = 0x1F

type realModelRuntime struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

type segtestCase struct {
	Name     string
	Desc     string
	Tier     string
	Group    string
	Path     string
	Messages []memory.Message
}

type segtestCorpus struct {
	Groups        map[string][]segtestCase
	TotalCases    int
	TotalMessages int
}

type scenarioMeta struct {
	Name          string
	TotalMessages int
	Conversations int
}

func requireRealModelRuntime(t *testing.T) realModelRuntime {
	t.Helper()

	apiKey := strings.TrimSpace(os.Getenv("MEMORY_IT_API_KEY"))
	provider := "memory_it"
	if apiKey == "" {
		if qwen := strings.TrimSpace(os.Getenv("QWEN_API_KEY")); qwen != "" {
			apiKey = qwen
			provider = "qwen"
		}
	}
	if apiKey == "" {
		if openai := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); openai != "" {
			apiKey = openai
			provider = "openai"
		}
	}
	if apiKey == "" {
		t.Skip("skip real-model integration: set MEMORY_IT_API_KEY or QWEN_API_KEY or OPENAI_API_KEY")
	}

	baseURL := strings.TrimSpace(os.Getenv("MEMORY_IT_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("MEMORY_IT_MODEL"))

	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
			if model == "" {
				model = "gpt-4o-mini"
			}
		default:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
			if model == "" {
				model = "qwen-turbo-latest"
			}
		}
	} else if model == "" {
		model = "qwen-turbo-latest"
	}

	timeout := 45 * time.Second
	if raw := strings.TrimSpace(os.Getenv("MEMORY_IT_TIMEOUT_SEC")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}

	return realModelRuntime{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: timeout,
	}
}

func loadSegtestCorpus(t *testing.T) *segtestCorpus {
	t.Helper()

	root := filepath.Join("testdata", "corpus", "seg_cases")
	groups := []string{"simple", "complex", "long"}
	out := &segtestCorpus{Groups: make(map[string][]segtestCase, len(groups))}

	for _, group := range groups {
		pattern := filepath.Join(root, group, "*.yaml")
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		sort.Strings(files)
		if len(files) == 0 {
			t.Fatalf("no dataset files found for group %s", group)
		}

		for _, file := range files {
			parsed, err := parseSegtestCase(file, group)
			if err != nil {
				t.Fatalf("parse segtest case %s: %v", file, err)
			}
			out.Groups[group] = append(out.Groups[group], parsed)
			out.TotalCases++
			out.TotalMessages += len(parsed.Messages)
		}
	}

	return out
}

func listScenarioNames(t *testing.T) []string {
	t.Helper()

	root := filepath.Join("testdata", "corpus", "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read scenario dir %s: %v", root, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "m") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("no memory scenarios found in %s", root)
	}
	return names
}

func loadScenarioMeta(t *testing.T, scenario string) scenarioMeta {
	t.Helper()

	metaPath := filepath.Join("testdata", "corpus", "scenarios", scenario, "meta.yaml")
	content, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta %s: %v", metaPath, err)
	}

	meta := scenarioMeta{Name: scenario}
	for _, line := range strings.Split(string(content), "\n") {
		s := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(s, "total_messages:"):
			raw := strings.TrimSpace(strings.TrimPrefix(s, "total_messages:"))
			v, err := strconv.Atoi(raw)
			if err != nil {
				t.Fatalf("parse total_messages in %s: %v", metaPath, err)
			}
			meta.TotalMessages = v
		case strings.HasPrefix(s, "conversations:"):
			raw := strings.TrimSpace(strings.TrimPrefix(s, "conversations:"))
			v, err := strconv.Atoi(raw)
			if err != nil {
				t.Fatalf("parse conversations in %s: %v", metaPath, err)
			}
			meta.Conversations = v
		}
	}

	if meta.TotalMessages <= 0 {
		t.Fatalf("meta %s has invalid total_messages=%d", metaPath, meta.TotalMessages)
	}
	if meta.Conversations <= 0 {
		t.Fatalf("meta %s has invalid conversations=%d", metaPath, meta.Conversations)
	}

	return meta
}

func sumScenarioMessagesByMeta(t *testing.T) int {
	t.Helper()

	total := 0
	for _, name := range listScenarioNames(t) {
		meta := loadScenarioMeta(t, name)
		total += meta.TotalMessages
	}
	return total
}

func loadScenarioMessages(t *testing.T, scenario string, maxMessages int) []memory.Message {
	t.Helper()

	pattern := filepath.Join("testdata", "corpus", "scenarios", scenario, "conv_*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no conversation files for scenario %s", scenario)
	}

	all := make([]memory.Message, 0, 1024)
	ts := int64(1_000_000_000)
	for _, file := range files {
		msgs, nextTS, err := parseScenarioConversationFile(file, ts)
		if err != nil {
			t.Fatalf("parse conversation file %s: %v", file, err)
		}
		ts = nextTS
		all = append(all, msgs...)
		if maxMessages > 0 && len(all) >= maxMessages {
			all = all[:maxMessages]
			break
		}
	}

	if len(all) == 0 {
		t.Fatalf("scenario %s has zero messages", scenario)
	}
	return all
}

func parseScenarioConversationFile(path string, tsStart int64) ([]memory.Message, int64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, tsStart, err
	}

	lines := strings.Split(string(content), "\n")
	messages := make([]memory.Message, 0, 32)
	ts := tsStart
	inMessages := false
	var cur memory.Message
	hasCur := false

	flush := func() {
		if !hasCur {
			return
		}
		if strings.TrimSpace(cur.Content) == "" {
			hasCur = false
			cur = memory.Message{}
			return
		}
		cur.Timestamp = ts
		ts += 1_000_000
		messages = append(messages, cur)
		hasCur = false
		cur = memory.Message{}
	}

	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}

		if strings.HasPrefix(s, "messages:") {
			inMessages = true
			continue
		}
		if !inMessages {
			continue
		}

		if strings.HasPrefix(s, "- role:") {
			flush()
			rawRole := strings.TrimSpace(strings.TrimPrefix(s, "- role:"))
			roleText, err := parseYAMLScalar(rawRole)
			if err != nil {
				return nil, tsStart, fmt.Errorf("parse role scalar: %w", err)
			}
			cur = memory.Message{Role: parseRoleName(roleText)}
			hasCur = true
			continue
		}

		if !hasCur {
			continue
		}

		if strings.HasPrefix(s, "name:") {
			rawName := strings.TrimSpace(strings.TrimPrefix(s, "name:"))
			name, err := parseYAMLScalar(rawName)
			if err != nil {
				return nil, tsStart, fmt.Errorf("parse name scalar: %w", err)
			}
			cur.Name = name
			continue
		}

		if strings.HasPrefix(s, "content:") {
			rawContent := strings.TrimSpace(strings.TrimPrefix(s, "content:"))
			body, err := parseYAMLScalar(rawContent)
			if err != nil {
				return nil, tsStart, fmt.Errorf("parse content scalar: %w", err)
			}
			cur.Content = body
			continue
		}
	}

	flush()
	return messages, ts, nil
}

func parseSegtestCase(path, group string) (segtestCase, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return segtestCase{}, err
	}

	lines := strings.Split(string(content), "\n")
	parsed := segtestCase{Group: group, Path: path}
	var ts int64 = 1_000_000_000
	inMessages := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "name:"):
			parsed.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			inMessages = false
			continue
		case strings.HasPrefix(trimmed, "desc:"):
			raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "desc:"))
			text, err := parseYAMLScalar(raw)
			if err != nil {
				return segtestCase{}, fmt.Errorf("line %d parse desc: %w", i+1, err)
			}
			parsed.Desc = text
			inMessages = false
			continue
		case strings.HasPrefix(trimmed, "tier:"):
			parsed.Tier = strings.TrimSpace(strings.TrimPrefix(trimmed, "tier:"))
			inMessages = false
			continue
		case strings.HasPrefix(trimmed, "messages:"):
			inMessages = true
			continue
		}

		if !inMessages {
			continue
		}

		if !strings.HasPrefix(trimmed, "- ") {
			inMessages = false
			continue
		}

		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		text, err := parseYAMLScalar(raw)
		if err != nil {
			return segtestCase{}, fmt.Errorf("line %d parse message: %w", i+1, err)
		}

		role, body := parseRoleAndContent(text)
		parsed.Messages = append(parsed.Messages, memory.Message{
			Role:      role,
			Content:   body,
			Timestamp: ts,
		})
		ts += 1_000_000
	}

	if parsed.Name == "" {
		parsed.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if parsed.Tier == "" {
		parsed.Tier = group
	}
	if len(parsed.Messages) == 0 {
		return segtestCase{}, fmt.Errorf("case %s has no messages", path)
	}
	return parsed, nil
}

func parseYAMLScalar(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		text, err := strconv.Unquote(raw)
		if err != nil {
			return "", err
		}
		return text, nil
	}

	if strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2 {
		text := raw[1 : len(raw)-1]
		text = strings.ReplaceAll(text, "''", "'")
		return text, nil
	}

	return raw, nil
}

func parseRoleAndContent(text string) (memory.Role, string) {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) < 2 {
		return memory.RoleUser, strings.TrimSpace(text)
	}

	roleName := strings.ToLower(strings.TrimSpace(parts[0]))
	content := strings.TrimSpace(parts[1])
	return parseRoleName(roleName), content
}

func parseRoleName(roleName string) memory.Role {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	switch roleName {
	case "assistant", "model":
		return memory.RoleModel
	case "tool":
		return memory.RoleTool
	case "user":
		fallthrough
	default:
		return memory.RoleUser
	}
}

func chunkMessages(messages []memory.Message, chunkSize int) [][]memory.Message {
	if chunkSize <= 0 {
		return nil
	}
	chunks := make([][]memory.Message, 0, (len(messages)+chunkSize-1)/chunkSize)
	for i := 0; i < len(messages); i += chunkSize {
		j := i + chunkSize
		if j > len(messages) {
			j = len(messages)
		}
		part := make([]memory.Message, j-i)
		copy(part, messages[i:j])
		chunks = append(chunks, part)
	}
	return chunks
}

func newIntegrationHost(t *testing.T, compressor memory.Compressor) *memory.Host {
	t.Helper()

	store := kv.NewMemory(&kv.Options{Separator: integrationSeparator})
	host, err := memory.NewHost(context.Background(), memory.HostConfig{
		Store:          store,
		Compressor:     compressor,
		CompressPolicy: memory.CompressPolicy{MaxMessages: 1 << 30, MaxChars: 1 << 30},
		Separator:      integrationSeparator,
	})
	if err != nil {
		t.Fatalf("new integration host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close() })
	return host
}

func graphEntityCount(ctx context.Context, mem *memory.Memory) (int, error) {
	count := 0
	for _, err := range mem.Graph().ListEntities(ctx, "") {
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

func ingestMessagesWithGroup(ctx context.Context, mem *memory.Memory, compressor memory.Compressor, messages []memory.Message, groupLabel string) error {
	result, err := compressor.CompressMessages(ctx, messages)
	if err != nil {
		return err
	}

	for _, seg := range result.Segments {
		seg.Labels = appendUnique(seg.Labels, groupLabel)
		if err := mem.StoreSegment(ctx, seg, recall.Bucket1H); err != nil {
			return err
		}
	}

	update, err := compressor.ExtractEntities(ctx, messages)
	if err != nil {
		return err
	}
	if update == nil {
		update = &memory.EntityUpdate{}
	}
	update.Entities = append(update.Entities, memory.EntityInput{
		Label: groupLabel,
		Attrs: map[string]any{"group": groupLabel},
	})

	return mem.ApplyEntityUpdate(ctx, update)
}

func appendUnique(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

func segmentHasLabel(seg memory.ScoredSegment, label string) bool {
	for _, l := range seg.Labels {
		if l == label {
			return true
		}
	}
	return false
}

func allSegmentsHaveLabel(segments []memory.ScoredSegment, label string) bool {
	if len(segments) == 0 {
		return false
	}
	for _, seg := range segments {
		if !segmentHasLabel(seg, label) {
			return false
		}
	}
	return true
}

func anySegmentHasLabel(segments []memory.ScoredSegment, label string) bool {
	for _, seg := range segments {
		if segmentHasLabel(seg, label) {
			return true
		}
	}
	return false
}

func failOrSkipTransient(t *testing.T, phase string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if isTransientError(err) {
		t.Skipf("skip due transient error at %s: %v", phase, err)
	}
	t.Fatalf("%s: %v", phase, err)
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporarily unavailable") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "status 429") ||
		strings.Contains(msg, "status 502") ||
		strings.Contains(msg, "status 503") {
		return true
	}
	return false
}

type openAICompatClient struct {
	runtime    realModelRuntime
	httpClient *http.Client
}

func newOpenAICompatClient(runtime realModelRuntime) *openAICompatClient {
	return &openAICompatClient{
		runtime: runtime,
		httpClient: &http.Client{
			Timeout: runtime.Timeout,
		},
	}
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	Temperature    float64           `json:"temperature,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *openAICompatClient) chatJSON(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	content, err := c.chat(ctx, systemPrompt, userPrompt, maxTokens, true)
	if err == nil {
		return content, nil
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "response_format") || strings.Contains(lower, "json_object") {
		return c.chat(ctx, systemPrompt, userPrompt, maxTokens, false)
	}

	return "", err
}

func (c *openAICompatClient) chat(ctx context.Context, systemPrompt, userPrompt string, maxTokens int, jsonMode bool) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 1000
	}

	reqBody := chatRequest{
		Model: c.runtime.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0,
		MaxTokens:   maxTokens,
	}
	if jsonMode {
		reqBody.ResponseFormat = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(c.runtime.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.runtime.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("chat status %d: %s", resp.StatusCode, shorten(string(raw), 240))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("chat response has empty choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("chat response content is empty")
	}
	return content, nil
}

func decodeModelJSON(raw string, out any) error {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	if err := json.Unmarshal([]byte(clean), out); err == nil {
		return nil
	}

	start := strings.IndexByte(clean, '{')
	end := strings.LastIndexByte(clean, '}')
	if start >= 0 && end > start {
		obj := clean[start : end+1]
		if err := json.Unmarshal([]byte(obj), out); err == nil {
			return nil
		}
	}

	return fmt.Errorf("decode model json failed, raw=%s", shorten(clean, 240))
}

func shorten(text string, limit int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

type realModelCompressor struct {
	client *openAICompatClient
}

func newRealModelCompressor(runtime realModelRuntime) *realModelCompressor {
	return &realModelCompressor{client: newOpenAICompatClient(runtime)}
}

type segmentModelOutput struct {
	Summary  string   `json:"summary"`
	Keywords []string `json:"keywords"`
	Labels   []string `json:"labels"`
}

type profileModelOutput struct {
	Entities []struct {
		Label string         `json:"label"`
		Attrs map[string]any `json:"attrs"`
	} `json:"entities"`
	Relations []struct {
		From    string `json:"from"`
		To      string `json:"to"`
		RelType string `json:"rel_type"`
	} `json:"relations"`
}

func (c *realModelCompressor) CompressMessages(ctx context.Context, messages []memory.Message) (*memory.CompressResult, error) {
	if len(messages) == 0 {
		return &memory.CompressResult{}, nil
	}

	sampled := sampleModelMessages(messages, 220)
	seg, err := c.segment(ctx, formatMessageLines(sampled, 40_000), 1200)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(seg.Summary) == "" {
		seg.Summary = strings.TrimSpace(sampled[len(sampled)-1].Content)
	}
	if len(seg.Keywords) == 0 {
		seg.Keywords = []string{"memory", "integration"}
	}
	if len(seg.Labels) == 0 {
		seg.Labels = []string{"topic:记忆"}
	}

	return &memory.CompressResult{
		Segments: []memory.SegmentInput{{
			Summary:  seg.Summary,
			Keywords: dedupeStrings(seg.Keywords),
			Labels:   dedupeStrings(seg.Labels),
		}},
		Summary: seg.Summary,
	}, nil
}

func (c *realModelCompressor) ExtractEntities(ctx context.Context, messages []memory.Message) (*memory.EntityUpdate, error) {
	if len(messages) == 0 {
		return &memory.EntityUpdate{}, nil
	}

	sampled := sampleModelMessages(messages, 220)
	systemPrompt := `你是对话画像器（profile）。
请从对话里抽取人物/主题实体和关系，返回严格 JSON：
{
  "entities": [{"label": "person:小明", "attrs": {"likes": "恐龙"}}],
  "relations": [{"from": "person:小明", "to": "topic:恐龙", "rel_type": "likes"}]
}
要求：
1) label 必须是字符串；
2) attrs 为对象；
3) 只返回 JSON，不要解释。`

	userPrompt := "对话如下：\n" + formatMessageLines(sampled, 40_000)
	raw, err := c.client.chatJSON(ctx, systemPrompt, userPrompt, 1200)
	if err != nil {
		return nil, err
	}

	var parsed profileModelOutput
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return nil, err
	}

	update := &memory.EntityUpdate{}
	for _, e := range parsed.Entities {
		label := strings.TrimSpace(e.Label)
		if label == "" {
			continue
		}
		update.Entities = append(update.Entities, memory.EntityInput{Label: label, Attrs: e.Attrs})
	}
	for _, r := range parsed.Relations {
		from := strings.TrimSpace(r.From)
		to := strings.TrimSpace(r.To)
		relType := strings.TrimSpace(r.RelType)
		if from == "" || to == "" || relType == "" {
			continue
		}
		update.Relations = append(update.Relations, memory.RelationInput{From: from, To: to, RelType: relType})
	}

	if len(update.Entities) == 0 && len(update.Relations) == 0 {
		seg, segErr := c.segment(ctx, formatMessageLines(sampled, 30_000), 900)
		if segErr == nil {
			for _, label := range dedupeStrings(seg.Labels) {
				update.Entities = append(update.Entities, memory.EntityInput{
					Label: label,
					Attrs: map[string]any{"source": "segment-fallback"},
				})
			}
			if len(seg.Labels) >= 2 {
				update.Relations = append(update.Relations, memory.RelationInput{
					From:    seg.Labels[0],
					To:      seg.Labels[1],
					RelType: "related_to",
				})
			}
		}
	}

	return update, nil
}

func (c *realModelCompressor) CompactSegments(ctx context.Context, summaries []string) (*memory.CompressResult, error) {
	if len(summaries) == 0 {
		return &memory.CompressResult{}, nil
	}

	lines := make([]string, 0, len(summaries))
	for i, summary := range summaries {
		summary = strings.TrimSpace(summary)
		if summary == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("summary-%d: %s", i+1, summary))
		if len(lines) >= 120 {
			break
		}
	}
	if len(lines) == 0 {
		return &memory.CompressResult{}, nil
	}

	seg, err := c.segment(ctx, strings.Join(lines, "\n"), 1000)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(seg.Summary) == "" {
		seg.Summary = strings.Join(lines, " | ")
	}

	return &memory.CompressResult{
		Segments: []memory.SegmentInput{{
			Summary:  seg.Summary,
			Keywords: dedupeStrings(seg.Keywords),
			Labels:   dedupeStrings(seg.Labels),
		}},
		Summary: seg.Summary,
	}, nil
}

func (c *realModelCompressor) segment(ctx context.Context, messageText string, maxTokens int) (*segmentModelOutput, error) {
	systemPrompt := `你是对话分段器（segmentation）。
请把输入整理成一个记忆片段，并返回严格 JSON：
{
  "summary": "一句或两句总结",
  "keywords": ["关键词1", "关键词2"],
  "labels": ["person:小明", "topic:恐龙"]
}
要求：
1) summary 尽量覆盖主要事实；
2) keywords 3~10 个；
3) labels 至少 2 个，优先 person/topic 命名空间；
4) 只输出 JSON。`

	userPrompt := "对话如下：\n" + messageText
	raw, err := c.client.chatJSON(ctx, systemPrompt, userPrompt, maxTokens)
	if err != nil {
		return nil, err
	}

	var parsed segmentModelOutput
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return nil, err
	}

	parsed.Summary = strings.TrimSpace(parsed.Summary)
	parsed.Keywords = dedupeStrings(parsed.Keywords)
	parsed.Labels = dedupeStrings(parsed.Labels)
	return &parsed, nil
}

func sampleModelMessages(messages []memory.Message, limit int) []memory.Message {
	if limit <= 0 || len(messages) <= limit {
		out := make([]memory.Message, len(messages))
		copy(out, messages)
		return out
	}

	part := limit / 3
	if part < 1 {
		part = 1
	}

	begin := make([]memory.Message, 0, part)
	begin = append(begin, messages[:part]...)

	midStart := len(messages)/2 - part/2
	if midStart < part {
		midStart = part
	}
	midEnd := midStart + part
	if midEnd > len(messages)-part {
		midEnd = len(messages) - part
		if midEnd < midStart {
			midEnd = midStart
		}
	}
	middle := messages[midStart:midEnd]

	endStart := len(messages) - part
	if endStart < 0 {
		endStart = 0
	}
	end := messages[endStart:]

	out := make([]memory.Message, 0, len(begin)+len(middle)+len(end))
	out = append(out, begin...)
	out = append(out, middle...)
	out = append(out, end...)
	return out
}

func formatMessageLines(messages []memory.Message, maxChars int) string {
	if len(messages) == 0 {
		return ""
	}

	var b strings.Builder
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		line := fmt.Sprintf("- %s: %s\n", msg.Role, content)
		if maxChars > 0 && b.Len()+len(line) > maxChars {
			break
		}
		b.WriteString(line)
	}
	return b.String()
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
