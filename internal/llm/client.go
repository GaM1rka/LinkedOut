package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const resultSystemPrompt = "Ты карьерный ассистент для студентов и junior IT-специалистов. Твоя задача — помочь честно и конкретно упаковать проектный опыт в резюме. Не выдумывай факты, метрики, технологии и достижения. Не делай формулировки слишком senior, если пользователь не подтвердил такой уровень ответственности. Используй только информацию из ответов пользователя. Если данных недостаточно, пиши осторожно и без выдуманных цифр. Итог должен быть полезен для резюме и собеседования."

const nextQuestionSystemPrompt = "Задай один короткий уточняющий вопрос по проекту. Вопрос должен помочь лучше понять личный вклад, стек, сложность, результат или то, что пользователь сможет защитить на интервью. Не повторяй уже заданные вопросы. Не задавай больше одного вопроса."

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	log        *slog.Logger
}

func NewClient(apiKey, baseURL, model string, httpClient *http.Client, log *slog.Logger) *Client {
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:      strings.TrimSpace(model),
		httpClient: httpClient,
		log:        log,
	}
}

type QA struct {
	Order    int
	Question string
	Answer   string
}

type InterviewContext struct {
	TargetRole        string
	ProjectType       string
	NextQuestionOrder int
	MaxQuestions      int
	RequiredFocus     string
	Answers           []QA
}

type GeneratedResult struct {
	Bullets        string `json:"bullets"`
	ProjectSummary string `json:"project_summary"`
	Skills         string `json:"skills"`
	RiskWarnings   string `json:"risk_warnings"`
	DontWrite      string `json:"dont_write"`
	RawResponse    string `json:"raw_response"`
}

func (c *Client) GenerateNextQuestion(ctx context.Context, interview InterviewContext) (string, error) {
	userPrompt := fmt.Sprintf(`Контекст интервью:
Целевая роль: %s
Тип проекта: %s
Вопрос номер: %d из %d
Следующий обязательный аспект, который нужно покрыть: %s

Уже заданные вопросы и ответы:
%s

Сформулируй один следующий вопрос по-русски. Вопрос должен быть коротким и конкретным.`,
		interview.TargetRole,
		interview.ProjectType,
		interview.NextQuestionOrder,
		interview.MaxQuestions,
		interview.RequiredFocus,
		compactAnswers(interview.Answers, 5000),
	)

	content, err := c.chat(ctx, []message{
		{Role: "system", Content: nextQuestionSystemPrompt},
		{Role: "user", Content: userPrompt},
	}, 180, 0.2)
	if err != nil {
		return "", err
	}
	return cleanupQuestion(content), nil
}

func (c *Client) GenerateResumeDescription(ctx context.Context, interview InterviewContext) (GeneratedResult, error) {
	userPrompt := fmt.Sprintf(`Контекст интервью:
Целевая роль: %s
Тип проекта: %s

Ответы пользователя:
%s

Сгенерируй результат строго в JSON без markdown:
{
  "bullets": "3-5 bullet points для резюме, каждый с новой строки",
  "project_summary": "краткое описание проекта",
  "skills": "список подтверждённых навыков",
  "risk_warnings": "что звучит рискованно: вода, неподтверждённые метрики, слишком senior-формулировки",
  "dont_write": "что лучше не писать, если пользователь не сможет защитить это на интервью"
}

Требования:
- честно и конкретно;
- без выдуманных метрик, технологий, денег, пользователей или процентов;
- без senior-формулировок, если ответственность не подтверждена;
- используй только ответы пользователя.`,
		interview.TargetRole,
		interview.ProjectType,
		compactAnswers(interview.Answers, 7000),
	)

	raw, err := c.chat(ctx, []message{
		{Role: "system", Content: resultSystemPrompt},
		{Role: "user", Content: userPrompt},
	}, 1200, 0.2)
	if err != nil {
		return GeneratedResult{}, err
	}

	var result GeneratedResult
	if err := json.Unmarshal([]byte(stripJSONFence(raw)), &result); err != nil {
		c.log.Warn("failed to parse llm JSON result, using raw response fallback", "error", err)
		result = GeneratedResult{
			Bullets:        strings.TrimSpace(raw),
			ProjectSummary: "LLM вернула ответ не в JSON-формате. Проверь raw response.",
			Skills:         "",
			RiskWarnings:   "",
		}
	}
	result.RawResponse = raw
	return result, nil
}

func (c *Client) chat(ctx context.Context, messages []message, maxTokens int, temperature float64) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("LLM_API_KEY is empty")
	}
	if c.baseURL == "" {
		return "", errors.New("LLM_BASE_URL is empty")
	}
	if c.model == "" {
		return "", errors.New("LLM_MODEL is empty")
	}

	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, strings.TrimSpace(string(errorBody)))
	}

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", errors.New("llm response has no choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("llm response is empty")
	}
	return content, nil
}

func (c *Client) chatURL() string {
	if strings.HasSuffix(c.baseURL, "/chat/completions") {
		return c.baseURL
	}
	return c.baseURL + "/chat/completions"
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func compactAnswers(answers []QA, maxRunes int) string {
	if len(answers) == 0 {
		return "Пока нет ответов."
	}

	var b strings.Builder
	for _, qa := range answers {
		fmt.Fprintf(&b, "%d. Вопрос: %s\nОтвет: %s\n\n", qa.Order, qa.Question, qa.Answer)
	}
	return truncateRunes(strings.TrimSpace(b.String()), maxRunes)
}

func truncateRunes(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "\n...[контекст сокращён]"
}

func cleanupQuestion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	lines := strings.Split(value, "\n")
	if len(lines) > 0 {
		value = strings.TrimSpace(lines[0])
	}
	if value == "" {
		return ""
	}
	if !strings.HasSuffix(value, "?") {
		value += "?"
	}
	return value
}

func stripJSONFence(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}
