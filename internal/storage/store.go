package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"linkedout/internal/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

type User struct {
	ID         string    `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type UpsertUserInput struct {
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
	FirstName  string `json:"first_name"`
}

func (s *Store) UpsertUser(ctx context.Context, input UpsertUserInput) (User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (telegram_id, username, first_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_id) DO UPDATE
		SET username = CASE WHEN EXCLUDED.username <> '' THEN EXCLUDED.username ELSE users.username END,
		    first_name = CASE WHEN EXCLUDED.first_name <> '' THEN EXCLUDED.first_name ELSE users.first_name END
		RETURNING id::text, telegram_id, COALESCE(username, ''), COALESCE(first_name, ''), created_at
	`, input.TelegramID, input.Username, input.FirstName).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.CreatedAt,
	)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

type StartInterviewInput struct {
	TelegramID  int64  `json:"telegram_id"`
	TargetRole  string `json:"target_role"`
	ProjectType string `json:"project_type"`
}

type InterviewStarted struct {
	InterviewID string    `json:"interview_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
}

func (s *Store) StartInterview(ctx context.Context, input StartInterviewInput) (InterviewStarted, error) {
	user, err := s.UpsertUser(ctx, UpsertUserInput{TelegramID: input.TelegramID})
	if err != nil {
		return InterviewStarted{}, err
	}

	var out InterviewStarted
	err = s.db.QueryRow(ctx, `
		INSERT INTO interviews (user_id, target_role, project_type, status)
		VALUES ($1::uuid, $2, $3, 'started')
		RETURNING id::text, status, started_at
	`, user.ID, input.TargetRole, input.ProjectType).Scan(&out.InterviewID, &out.Status, &out.StartedAt)
	if err != nil {
		return InterviewStarted{}, fmt.Errorf("start interview: %w", err)
	}
	return out, nil
}

type AnswerInput struct {
	QuestionOrder int    `json:"question_order"`
	QuestionText  string `json:"question_text"`
	AnswerText    string `json:"answer_text"`
}

func (s *Store) AddAnswer(ctx context.Context, interviewID string, input AnswerInput) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add answer: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `
		INSERT INTO interview_answers (interview_id, question_order, question_text, answer_text)
		VALUES ($1::uuid, $2, $3, $4)
	`, interviewID, input.QuestionOrder, input.QuestionText, input.AnswerText); err != nil {
		return fmt.Errorf("insert answer: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE interviews SET status = 'in_progress', updated_at = now() WHERE id = $1::uuid`, interviewID); err != nil {
		return fmt.Errorf("update interview after answer: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add answer: %w", err)
	}
	return nil
}

type CompleteInterviewInput struct {
	CompletedAt *time.Time `json:"completed_at"`
}

type InterviewCompleted struct {
	InterviewID     string `json:"interview_id"`
	DurationSeconds int    `json:"duration_seconds"`
}

func (s *Store) CompleteInterview(ctx context.Context, interviewID string, input CompleteInterviewInput) (InterviewCompleted, error) {
	completedAt := time.Now().UTC()
	if input.CompletedAt != nil {
		completedAt = input.CompletedAt.UTC()
	}

	var out InterviewCompleted
	err := s.db.QueryRow(ctx, `
		UPDATE interviews
		SET status = 'completed',
		    completed_at = $2,
		    duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM ($2 - started_at))::int),
		    updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, duration_seconds
	`, interviewID, completedAt).Scan(&out.InterviewID, &out.DurationSeconds)
	if err != nil {
		return InterviewCompleted{}, fmt.Errorf("complete interview: %w", err)
	}
	return out, nil
}

type GenerationInput struct {
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

func (s *Store) CreateGeneration(ctx context.Context, input GenerationInput) (GenerationCreated, error) {
	var out GenerationCreated
	err := s.db.QueryRow(ctx, `
		INSERT INTO generations (interview_id, bullets, project_summary, skills, risk_warnings, raw_llm_response)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, input.InterviewID, input.Bullets, input.ProjectSummary, input.Skills, input.RiskWarnings, input.RawLLMResponse).Scan(&out.GenerationID)
	if err != nil {
		return GenerationCreated{}, fmt.Errorf("create generation: %w", err)
	}
	return out, nil
}

type FeedbackInput struct {
	InterviewID        string `json:"interview_id"`
	GenerationID       string `json:"generation_id"`
	ReadyToUse         bool   `json:"ready_to_use"`
	NeedsMajorEdits    bool   `json:"needs_major_edits"`
	IssueType          string `json:"issue_type"`
	PaymentWillingness string `json:"payment_willingness"`
	Comment            string `json:"comment"`
}

func (s *Store) CreateFeedback(ctx context.Context, input FeedbackInput) error {
	issueType := input.IssueType
	if issueType == "" {
		issueType = "none"
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO feedbacks (interview_id, generation_id, ready_to_use, needs_major_edits, issue_type, payment_willingness, comment)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
	`, input.InterviewID, input.GenerationID, input.ReadyToUse, input.NeedsMajorEdits, issueType, input.PaymentWillingness, input.Comment)
	if err != nil {
		return fmt.Errorf("create feedback: %w", err)
	}
	return nil
}

type ManualReviewInput struct {
	GenerationID       string `json:"generation_id"`
	ReviewerTelegramID int64  `json:"reviewer_telegram_id"`
	Status             string `json:"status"`
	Comment            string `json:"comment"`
}

func (s *Store) CreateManualReview(ctx context.Context, input ManualReviewInput) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO manual_reviews (generation_id, reviewer_telegram_id, status, comment)
		VALUES ($1::uuid, $2, $3, $4)
	`, input.GenerationID, input.ReviewerTelegramID, input.Status, input.Comment)
	if err != nil {
		return fmt.Errorf("create manual review: %w", err)
	}
	return nil
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

func (s *Store) ListPendingReviews(ctx context.Context, limit int) ([]PendingReview, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}

	rows, err := s.db.Query(ctx, `
		SELECT g.id::text,
		       u.telegram_id,
		       i.target_role,
		       i.project_type,
		       g.bullets,
		       g.project_summary,
		       g.created_at
		FROM generations g
		JOIN interviews i ON i.id = g.interview_id
		JOIN users u ON u.id = i.user_id
		WHERE NOT EXISTS (
		    SELECT 1 FROM manual_reviews mr WHERE mr.generation_id = g.id
		)
		ORDER BY g.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]PendingReview, 0)
	for rows.Next() {
		var review PendingReview
		if err := rows.Scan(
			&review.GenerationID,
			&review.TelegramID,
			&review.TargetRole,
			&review.ProjectType,
			&review.Bullets,
			&review.ProjectSummary,
			&review.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending reviews: %w", err)
	}

	return reviews, nil
}

func (s *Store) Summary(ctx context.Context) (metrics.Summary, error) {
	raw := metrics.RawStats{}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM interviews`).Scan(&raw.StartedInterviews); err != nil {
		return metrics.Summary{}, fmt.Errorf("count started interviews: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM interviews WHERE status = 'completed'`).Scan(&raw.CompletedInterviews); err != nil {
		return metrics.Summary{}, fmt.Errorf("count completed interviews: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COALESCE(AVG(duration_seconds)::float8, 0) FROM interviews WHERE status = 'completed' AND duration_seconds IS NOT NULL`).Scan(&raw.AvgInterviewDurationSeconds); err != nil {
		return metrics.Summary{}, fmt.Errorf("avg duration: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE ready_to_use) FROM feedbacks`).Scan(&raw.FeedbackTotal, &raw.ReadyToUseCount); err != nil {
		return metrics.Summary{}, fmt.Errorf("feedback stats: %w", err)
	}
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT u.telegram_id)
		FROM feedbacks f
		JOIN interviews i ON i.id = f.interview_id
		JOIN users u ON u.id = i.user_id
		WHERE f.payment_willingness IN ('one_project', 'full_resume')
	`).Scan(&raw.PaymentWillingUsers); err != nil {
		return metrics.Summary{}, fmt.Errorf("payment willingness stats: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'pass') FROM manual_reviews`).Scan(&raw.ManualReviewTotal, &raw.ManualReviewPassed); err != nil {
		return metrics.Summary{}, fmt.Errorf("manual review stats: %w", err)
	}

	return metrics.CalculateSummary(raw), nil
}

type EventInput struct {
	TelegramID  *int64          `json:"telegram_id"`
	InterviewID string          `json:"interview_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

func (s *Store) CreateEvent(ctx context.Context, input EventInput) error {
	var telegramID any
	if input.TelegramID != nil {
		telegramID = *input.TelegramID
	}

	var interviewID any
	if input.InterviewID != "" {
		interviewID = input.InterviewID
	}

	var payload any
	if len(input.Payload) > 0 {
		payload = string(input.Payload)
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO events (user_id, interview_id, event_type, payload)
		VALUES (
		    (SELECT id FROM users WHERE telegram_id = $1),
		    $2::uuid,
		    $3,
		    $4::jsonb
		)
	`, telegramID, interviewID, input.EventType, payload)
	if err != nil && input.InterviewID == "" {
		_, err = s.db.Exec(ctx, `
			INSERT INTO events (user_id, event_type, payload)
			VALUES ((SELECT id FROM users WHERE telegram_id = $1), $2, $3::jsonb)
		`, telegramID, input.EventType, payload)
	}
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}
