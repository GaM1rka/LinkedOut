package metricsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

func New(baseURL string, httpClient *http.Client, log *slog.Logger) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		log:        log,
	}
}

type User struct {
	ID         string `json:"id"`
	TelegramID int64  `json:"telegram_id"`
}

type UpsertUserRequest struct {
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
	FirstName  string `json:"first_name"`
}

func (c *Client) UpsertUser(ctx context.Context, request UpsertUserRequest) (User, error) {
	var out User
	err := c.post(ctx, "/api/v1/users/upsert", request, &out)
	return out, err
}

type StartInterviewRequest struct {
	TelegramID  int64  `json:"telegram_id"`
	TargetRole  string `json:"target_role"`
	ProjectType string `json:"project_type"`
}

type InterviewStarted struct {
	InterviewID string    `json:"interview_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
}

func (c *Client) StartInterview(ctx context.Context, request StartInterviewRequest) (InterviewStarted, error) {
	var out InterviewStarted
	err := c.post(ctx, "/api/v1/interviews/start", request, &out)
	return out, err
}

type AddAnswerRequest struct {
	QuestionOrder int    `json:"question_order"`
	QuestionText  string `json:"question_text"`
	AnswerText    string `json:"answer_text"`
}

func (c *Client) AddAnswer(ctx context.Context, interviewID string, request AddAnswerRequest) error {
	if strings.TrimSpace(interviewID) == "" {
		return nil
	}
	return c.post(ctx, "/api/v1/interviews/"+url.PathEscape(interviewID)+"/answers", request, nil)
}

func (c *Client) CompleteInterview(ctx context.Context, interviewID string) error {
	if strings.TrimSpace(interviewID) == "" {
		return nil
	}
	return c.post(ctx, "/api/v1/interviews/"+url.PathEscape(interviewID)+"/complete", map[string]any{}, nil)
}

type GenerationRequest struct {
	InterviewID    string `json:"interview_id"`
	Bullets        string `json:"bullets"`
	ProjectSummary string `json:"project_summary"`
	Skills         string `json:"skills"`
	RiskWarnings   string `json:"risk_warnings"`
	RawLLMResponse string `json:"raw_llm_response"`
}

type GenerationCreated struct {
	GenerationID string `json:"generation_id"`
}

func (c *Client) CreateGeneration(ctx context.Context, request GenerationRequest) (GenerationCreated, error) {
	if strings.TrimSpace(request.InterviewID) == "" {
		return GenerationCreated{}, nil
	}
	var out GenerationCreated
	err := c.post(ctx, "/api/v1/generations", request, &out)
	return out, err
}

type FeedbackRequest struct {
	InterviewID        string `json:"interview_id"`
	GenerationID       string `json:"generation_id"`
	ReadyToUse         bool   `json:"ready_to_use"`
	NeedsMajorEdits    bool   `json:"needs_major_edits"`
	IssueType          string `json:"issue_type"`
	PaymentWillingness string `json:"payment_willingness"`
	Comment            string `json:"comment"`
}

func (c *Client) CreateFeedback(ctx context.Context, request FeedbackRequest) error {
	if strings.TrimSpace(request.InterviewID) == "" || strings.TrimSpace(request.GenerationID) == "" {
		return nil
	}
	return c.post(ctx, "/api/v1/feedbacks", request, nil)
}

type ManualReviewRequest struct {
	GenerationID       string `json:"generation_id"`
	ReviewerTelegramID int64  `json:"reviewer_telegram_id"`
	Status             string `json:"status"`
	Comment            string `json:"comment"`
}

func (c *Client) CreateManualReview(ctx context.Context, request ManualReviewRequest) error {
	return c.post(ctx, "/api/v1/manual-reviews", request, nil)
}

type PendingReview struct {
	GenerationID   string    `json:"generation_id"`
	TelegramID     int64     `json:"telegram_id"`
	TargetRole     string    `json:"target_role"`
	ProjectType    string    `json:"project_type"`
	Bullets        string    `json:"bullets"`
	ProjectSummary string    `json:"project_summary"`
	CreatedAt      time.Time `json:"created_at"`
}

func (c *Client) PendingReviews(ctx context.Context, limit int) ([]PendingReview, error) {
	var out []PendingReview
	err := c.get(ctx, fmt.Sprintf("/api/v1/manual-reviews/pending?limit=%d", limit), &out)
	return out, err
}

type Summary struct {
	StartedInterviews           int     `json:"started_interviews"`
	CompletedInterviews         int     `json:"completed_interviews"`
	CompletionRate              float64 `json:"completion_rate"`
	AvgInterviewDurationSeconds float64 `json:"avg_interview_duration_seconds"`
	AvgInterviewDurationMinutes float64 `json:"avg_interview_duration_minutes"`
	ReadyToUseRate              float64 `json:"ready_to_use_rate"`
	ReadyToUseCount             int     `json:"ready_to_use_count"`
	PaymentWillingUsers         int     `json:"payment_willing_users"`
	ManualReviewTotal           int     `json:"manual_review_total"`
	ManualReviewPassed          int     `json:"manual_review_passed"`
	ManualReviewPassRate        float64 `json:"manual_review_pass_rate"`
	MVPSuccess                  bool    `json:"mvp_success"`
	NotEnoughData               bool    `json:"not_enough_data"`
}

func (c *Client) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	err := c.get(ctx, "/api/v1/metrics/summary", &out)
	return out, err
}

type EventRequest struct {
	TelegramID  *int64 `json:"telegram_id,omitempty"`
	InterviewID string `json:"interview_id,omitempty"`
	EventType   string `json:"event_type"`
	Payload     any    `json:"payload,omitempty"`
}

func (c *Client) CreateEvent(ctx context.Context, request EventRequest) error {
	return c.post(ctx, "/api/v1/events", request, nil)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, input any, out any) error {
	return c.do(ctx, http.MethodPost, path, input, out)
}

func (c *Client) do(ctx context.Context, method, path string, input any, out any) error {
	if c.baseURL == "" {
		return fmt.Errorf("metrics base URL is empty")
	}

	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("marshal metrics request: %w", err)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		c.log.Info("metrics request started", "method", method, "path", path, "attempt", attempt)
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create metrics request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("metrics status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
			continue
		}
		if out != nil && len(responseBody) > 0 {
			if err := json.Unmarshal(responseBody, out); err != nil {
				return fmt.Errorf("decode metrics response: %w", err)
			}
		}
		c.log.Info("metrics request completed", "method", method, "path", path, "attempt", attempt, "status_code", resp.StatusCode)
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("metrics request failed")
	}
	c.log.Warn("metrics request failed", "method", method, "path", path, "error", lastErr)
	return lastErr
}
