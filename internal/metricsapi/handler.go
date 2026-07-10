package metricsapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"linkedout/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Handler struct {
	store *storage.Store
	log   *slog.Logger
}

func NewRouter(store *storage.Store, log *slog.Logger) http.Handler {
	h := &Handler{store: store, log: log}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.health)
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/users/upsert", h.upsertUser)
		r.Post("/interviews/start", h.startInterview)
		r.Post("/interviews/{interview_id}/answers", h.addAnswer)
		r.Post("/interviews/{interview_id}/complete", h.completeInterview)
		r.Post("/generations", h.createGeneration)
		r.Post("/feedbacks", h.createFeedback)
		r.Post("/manual-reviews", h.createManualReview)
		r.Get("/manual-reviews/pending", h.pendingReviews)
		r.Get("/metrics/summary", h.metricsSummary)
		r.Post("/events", h.createEvent)
	})

	return r
}

// health godoc
// @Summary Healthcheck
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} errorResponse
// @Router /health [get]
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	h.logRequest("health check requested", r)
	if err := h.store.Ping(r.Context()); err != nil {
		h.log.Error("health check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}
	h.log.Info("health check succeeded", "method", r.Method, "path", r.URL.Path)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// upsertUser godoc
// @Summary Upsert Telegram user
// @Tags users
// @Accept json
// @Produce json
// @Param request body storage.UpsertUserInput true "User"
// @Success 200 {object} storage.User
// @Failure 400 {object} errorResponse
// @Router /api/v1/users/upsert [post]
func (h *Handler) upsertUser(w http.ResponseWriter, r *http.Request) {
	var input storage.UpsertUserInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.TelegramID == 0 {
		writeError(w, http.StatusBadRequest, "telegram_id is required")
		return
	}

	h.logRequest("upsert user request received", r, "telegram_id", input.TelegramID, "username", input.Username, "first_name", input.FirstName)
	user, err := h.store.UpsertUser(r.Context(), input)
	if err != nil {
		h.log.Error("upsert user failed", "error", err, "telegram_id", input.TelegramID)
		writeError(w, http.StatusInternalServerError, "failed to upsert user")
		return
	}
	h.log.Info("upsert user completed", "telegram_id", user.TelegramID, "user_id", user.ID)
	writeJSON(w, http.StatusOK, user)
}

// startInterview godoc
// @Summary Start interview
// @Tags interviews
// @Accept json
// @Produce json
// @Param request body storage.StartInterviewInput true "Interview start"
// @Success 200 {object} storage.InterviewStarted
// @Failure 400 {object} errorResponse
// @Router /api/v1/interviews/start [post]
func (h *Handler) startInterview(w http.ResponseWriter, r *http.Request) {
	var input storage.StartInterviewInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.TelegramID == 0 || strings.TrimSpace(input.TargetRole) == "" || strings.TrimSpace(input.ProjectType) == "" {
		writeError(w, http.StatusBadRequest, "telegram_id, target_role and project_type are required")
		return
	}

	h.logRequest("start interview request received", r, "telegram_id", input.TelegramID, "target_role", input.TargetRole, "project_type", input.ProjectType)
	interview, err := h.store.StartInterview(r.Context(), input)
	if err != nil {
		h.log.Error("start interview failed", "error", err, "telegram_id", input.TelegramID)
		writeError(w, http.StatusInternalServerError, "failed to start interview")
		return
	}
	h.log.Info("start interview completed", "telegram_id", input.TelegramID, "interview_id", interview.InterviewID)
	writeJSON(w, http.StatusOK, interview)
}

// addAnswer godoc
// @Summary Save interview answer
// @Tags interviews
// @Accept json
// @Produce json
// @Param interview_id path string true "Interview ID"
// @Param request body storage.AnswerInput true "Answer"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errorResponse
// @Router /api/v1/interviews/{interview_id}/answers [post]
func (h *Handler) addAnswer(w http.ResponseWriter, r *http.Request) {
	interviewID := chi.URLParam(r, "interview_id")
	var input storage.AnswerInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(interviewID) == "" || input.QuestionOrder <= 0 || strings.TrimSpace(input.QuestionText) == "" || strings.TrimSpace(input.AnswerText) == "" {
		writeError(w, http.StatusBadRequest, "interview_id, question_order, question_text and answer_text are required")
		return
	}

	h.logRequest("add answer request received", r, "interview_id", interviewID, "question_order", input.QuestionOrder)
	if err := h.store.AddAnswer(r.Context(), interviewID, input); err != nil {
		h.log.Error("add answer failed", "error", err, "interview_id", interviewID)
		writeError(w, http.StatusInternalServerError, "failed to add answer")
		return
	}
	h.log.Info("add answer completed", "interview_id", interviewID, "question_order", input.QuestionOrder)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// completeInterview godoc
// @Summary Complete interview
// @Tags interviews
// @Accept json
// @Produce json
// @Param interview_id path string true "Interview ID"
// @Param request body storage.CompleteInterviewInput false "Completion time"
// @Success 200 {object} storage.InterviewCompleted
// @Failure 400 {object} errorResponse
// @Router /api/v1/interviews/{interview_id}/complete [post]
func (h *Handler) completeInterview(w http.ResponseWriter, r *http.Request) {
	interviewID := chi.URLParam(r, "interview_id")
	if strings.TrimSpace(interviewID) == "" {
		writeError(w, http.StatusBadRequest, "interview_id is required")
		return
	}

	var input storage.CompleteInterviewInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	h.logRequest("complete interview request received", r, "interview_id", interviewID)
	completed, err := h.store.CompleteInterview(r.Context(), interviewID, input)
	if err != nil {
		h.log.Error("complete interview failed", "error", err, "interview_id", interviewID)
		writeError(w, http.StatusInternalServerError, "failed to complete interview")
		return
	}
	h.log.Info("complete interview completed", "interview_id", interviewID, "duration_seconds", completed.DurationSeconds)
	writeJSON(w, http.StatusOK, completed)
}

// createGeneration godoc
// @Summary Save generated resume description
// @Tags generations
// @Accept json
// @Produce json
// @Param request body storage.GenerationInput true "Generation"
// @Success 200 {object} storage.GenerationCreated
// @Failure 400 {object} errorResponse
// @Router /api/v1/generations [post]
func (h *Handler) createGeneration(w http.ResponseWriter, r *http.Request) {
	var input storage.GenerationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.InterviewID) == "" {
		writeError(w, http.StatusBadRequest, "interview_id is required")
		return
	}

	h.logRequest("create generation request received", r, "interview_id", input.InterviewID)
	generation, err := h.store.CreateGeneration(r.Context(), input)
	if err != nil {
		h.log.Error("create generation failed", "error", err, "interview_id", input.InterviewID)
		writeError(w, http.StatusInternalServerError, "failed to create generation")
		return
	}
	h.log.Info("create generation completed", "interview_id", input.InterviewID, "generation_id", generation.GenerationID)
	writeJSON(w, http.StatusOK, generation)
}

// createFeedback godoc
// @Summary Save user feedback
// @Tags feedbacks
// @Accept json
// @Produce json
// @Param request body storage.FeedbackInput true "Feedback"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errorResponse
// @Router /api/v1/feedbacks [post]
func (h *Handler) createFeedback(w http.ResponseWriter, r *http.Request) {
	var input storage.FeedbackInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.InterviewID) == "" || strings.TrimSpace(input.GenerationID) == "" || strings.TrimSpace(input.PaymentWillingness) == "" {
		writeError(w, http.StatusBadRequest, "interview_id, generation_id and payment_willingness are required")
		return
	}

	h.logRequest("create feedback request received", r, "interview_id", input.InterviewID, "generation_id", input.GenerationID)
	if err := h.store.CreateFeedback(r.Context(), input); err != nil {
		h.log.Error("create feedback failed", "error", err, "interview_id", input.InterviewID, "generation_id", input.GenerationID)
		writeError(w, http.StatusInternalServerError, "failed to create feedback")
		return
	}
	h.log.Info("create feedback completed", "interview_id", input.InterviewID, "generation_id", input.GenerationID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// createManualReview godoc
// @Summary Save manual review
// @Tags manual_reviews
// @Accept json
// @Produce json
// @Param request body storage.ManualReviewInput true "Manual review"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errorResponse
// @Router /api/v1/manual-reviews [post]
func (h *Handler) createManualReview(w http.ResponseWriter, r *http.Request) {
	var input storage.ManualReviewInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.GenerationID) == "" || input.ReviewerTelegramID == 0 || strings.TrimSpace(input.Status) == "" {
		writeError(w, http.StatusBadRequest, "generation_id, reviewer_telegram_id and status are required")
		return
	}

	h.logRequest("create manual review request received", r, "generation_id", input.GenerationID, "reviewer_telegram_id", input.ReviewerTelegramID, "status", input.Status)
	if err := h.store.CreateManualReview(r.Context(), input); err != nil {
		h.log.Error("create manual review failed", "error", err, "generation_id", input.GenerationID)
		writeError(w, http.StatusInternalServerError, "failed to create manual review")
		return
	}
	h.log.Info("create manual review completed", "generation_id", input.GenerationID, "status", input.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// pendingReviews godoc
// @Summary List pending manual reviews
// @Tags manual_reviews
// @Produce json
// @Param limit query int false "Limit"
// @Success 200 {array} storage.PendingReview
// @Router /api/v1/manual-reviews/pending [get]
func (h *Handler) pendingReviews(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "limit must be an integer")
			return
		}
		limit = parsed
	}

	h.logRequest("pending reviews requested", r, "limit", limit)
	reviews, err := h.store.ListPendingReviews(r.Context(), limit)
	if err != nil {
		h.log.Error("list pending reviews failed", "error", err, "limit", limit)
		writeError(w, http.StatusInternalServerError, "failed to list pending reviews")
		return
	}
	h.log.Info("pending reviews completed", "limit", limit, "count", len(reviews))
	writeJSON(w, http.StatusOK, reviews)
}

// metricsSummary godoc
// @Summary MVP metrics summary
// @Tags metrics
// @Produce json
// @Success 200 {object} metrics.Summary
// @Router /api/v1/metrics/summary [get]
func (h *Handler) metricsSummary(w http.ResponseWriter, r *http.Request) {
	h.logRequest("metrics summary requested", r)
	summary, err := h.store.Summary(r.Context())
	if err != nil {
		h.log.Error("metrics summary failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to calculate metrics")
		return
	}
	h.log.Info("metrics summary completed", "started_interviews", summary.StartedInterviews, "completed_interviews", summary.CompletedInterviews, "completion_rate", summary.CompletionRate, "ready_to_use_rate", summary.ReadyToUseRate, "mvp_success", summary.MVPSuccess)
	writeJSON(w, http.StatusOK, summary)
}

// createEvent godoc
// @Summary Save product event
// @Tags events
// @Accept json
// @Produce json
// @Param request body storage.EventInput true "Event"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errorResponse
// @Router /api/v1/events [post]
func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	var input storage.EventInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.EventType) == "" {
		writeError(w, http.StatusBadRequest, "event_type is required")
		return
	}

	h.logRequest("create event request received", r, "event_type", input.EventType, "telegram_id", input.TelegramID, "interview_id", input.InterviewID)
	if err := h.store.CreateEvent(r.Context(), input); err != nil {
		h.log.Error("create event failed", "error", err, "event_type", input.EventType)
		writeError(w, http.StatusInternalServerError, "failed to create event")
		return
	}
	h.log.Info("create event completed", "event_type", input.EventType, "telegram_id", input.TelegramID, "interview_id", input.InterviewID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type errorResponse struct {
	Error string `json:"error"`
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) logRequest(action string, r *http.Request, fields ...any) {
	args := append([]any{"method", r.Method, "path", r.URL.Path}, fields...)
	h.log.Info(action, args...)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
