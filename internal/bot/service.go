package bot

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"linkedout/internal/llm"
	"linkedout/internal/metricsclient"
)

type Service struct {
	api        *tgbotapi.BotAPI
	llm        *llm.Client
	metrics    *metricsclient.Client
	adminIDs   map[int64]struct{}
	log        *slog.Logger
	sessionsMu sync.Mutex
	sessions   map[int64]*Session
}

func NewService(api *tgbotapi.BotAPI, llmClient *llm.Client, metrics *metricsclient.Client, adminIDs []int64, log *slog.Logger) *Service {
	admins := make(map[int64]struct{}, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = struct{}{}
	}

	return &Service{
		api:      api,
		llm:      llmClient,
		metrics:  metrics,
		adminIDs: admins,
		log:      log,
		sessions: make(map[int64]*Session),
	}
}

func (s *Service) Run(ctx context.Context) error {
	updates := s.api.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Limit:   0,
		Timeout: 30,
	})

	for {
		select {
		case <-ctx.Done():
			s.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			s.handleUpdate(ctx, update)
		}
	}
}

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		if update.Message.IsCommand() {
			s.handleCommand(ctx, update.Message)
			return
		}
		s.handleText(ctx, update.Message)
		return
	}

	if update.CallbackQuery != nil {
		s.handleCallback(ctx, update.CallbackQuery)
	}
}

func (s *Service) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		s.handleStart(ctx, msg)
	case "cancel":
		interviewID := ""
		if session := s.getSession(msg.From.ID); session != nil {
			interviewID = session.InterviewID
		}
		s.cancelSession(msg.From.ID)
		s.trackEvent(ctx, msg.From.ID, interviewID, "interview_cancelled", nil)
		s.sendText(msg.Chat.ID, "Ок, остановил текущее интервью. Можно начать заново через /start.")
	case "stats":
		s.handleStats(ctx, msg)
	case "pending_reviews":
		s.handlePendingReviews(ctx, msg)
	case "review":
		s.handleReview(ctx, msg)
	default:
		s.sendText(msg.Chat.ID, "Я пока знаю команды /start, /cancel, /stats, /pending_reviews и /review.")
	}
}

func (s *Service) handleStart(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	s.log.Info("start command received", "telegram_id", userID, "username", msg.From.UserName)
	_, _ = s.metrics.UpsertUser(ctx, metricsclient.UpsertUserRequest{
		TelegramID: userID,
		Username:   msg.From.UserName,
		FirstName:  msg.From.FirstName,
	})
	s.trackEvent(ctx, userID, "", "start_command", map[string]any{"username": msg.From.UserName})

	text := "Привет! Я LinkedOut. За 10-15 минут помогу упаковать твой проект в описание для резюме. Я задам несколько вопросов, а потом соберу bullet points, summary и список навыков."
	s.sendWithKeyboard(msg.Chat.ID, text, startKeyboard())
}

func (s *Service) handleText(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)

	session := s.getSession(userID)
	if session == nil || session.State == StateIdle || session.State == StateCompleted {
		s.sendText(msg.Chat.ID, "Чтобы начать интервью, отправь /start и нажми «Начать интервью».")
		return
	}

	if text == "" {
		s.sendText(msg.Chat.ID, "Ответ не должен быть пустым. Напиши коротко, своими словами.")
		return
	}

	switch session.State {
	case StateInterviewing:
		s.handleInterviewAnswer(ctx, msg, session, text)
	case StateWaitingTextFeedback:
		s.handleTextFeedback(ctx, msg, session, text)
	default:
		s.sendText(msg.Chat.ID, "Сейчас удобнее выбрать один из вариантов на кнопках.")
	}
}

func (s *Service) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	_, _ = s.api.Request(tgbotapi.NewCallback(callback.ID, ""))
	if callback.Message == nil {
		s.log.Warn("callback without message", "telegram_id", callback.From.ID, "data", callback.Data)
		return
	}

	userID := callback.From.ID
	chatID := callback.Message.Chat.ID
	data := callback.Data

	switch {
	case data == "start_interview":
		s.startInterviewFlow(ctx, userID, chatID)
	case strings.HasPrefix(data, "role|"):
		s.chooseRole(userID, chatID, strings.TrimPrefix(data, "role|"))
	case strings.HasPrefix(data, "ptype|"):
		s.chooseProjectType(ctx, userID, chatID, strings.TrimPrefix(data, "ptype|"))
	case strings.HasPrefix(data, "ready|"):
		s.handleReadyFeedback(userID, chatID, strings.TrimPrefix(data, "ready|"))
	case strings.HasPrefix(data, "issue|"):
		s.handleIssueFeedback(userID, chatID, strings.TrimPrefix(data, "issue|"))
	case strings.HasPrefix(data, "pay|"):
		s.handlePaymentFeedback(userID, chatID, strings.TrimPrefix(data, "pay|"))
	default:
		s.sendText(chatID, "Не понял действие. Попробуй /start.")
	}
}

func (s *Service) startInterviewFlow(ctx context.Context, userID int64, chatID int64) {
	if active := s.getSession(userID); active != nil && active.State != StateIdle && active.State != StateCompleted {
		s.sendText(chatID, "У тебя уже идёт интервью. Ответь на текущий вопрос или отправь /cancel, чтобы начать заново.")
		return
	}

	s.setSession(userID, &Session{
		TelegramID: userID,
		State:      StateChoosingRole,
		StartedAt:  time.Now().UTC(),
	})
	s.log.Info("interview flow started", "telegram_id", userID, "chat_id", chatID)
	s.trackEvent(ctx, userID, "", "interview_start_clicked", nil)
	s.sendWithKeyboard(chatID, "Для какой роли упаковываем проект?", roleKeyboard())
}

func (s *Service) chooseRole(userID int64, chatID int64, role string) {
	session := s.getSession(userID)
	if session == nil || session.State != StateChoosingRole {
		s.sendText(chatID, "Начни интервью через /start.")
		return
	}
	session.TargetRole = role
	session.State = StateChoosingProjectType
	s.sendWithKeyboard(chatID, "Какой это тип проекта?", projectTypeKeyboard())
}

func (s *Service) chooseProjectType(ctx context.Context, userID int64, chatID int64, projectType string) {
	session := s.getSession(userID)
	if session == nil || session.State != StateChoosingProjectType {
		s.sendText(chatID, "Начни интервью через /start.")
		return
	}

	session.ProjectType = projectType
	started, err := s.metrics.StartInterview(ctx, metricsclient.StartInterviewRequest{
		TelegramID:  userID,
		TargetRole:  session.TargetRole,
		ProjectType: session.ProjectType,
	})
	if err != nil {
		s.log.Warn("failed to start interview in metrics-service", "telegram_id", userID, "error", err)
	} else {
		session.InterviewID = started.InterviewID
	}

	session.State = StateInterviewing
	session.CurrentQuestionOrder = 1
	session.StartedAt = time.Now().UTC()
	s.trackEvent(ctx, userID, session.InterviewID, "interview_started", map[string]any{
		"target_role":  session.TargetRole,
		"project_type": session.ProjectType,
	})

	s.askQuestion(chatID, session, fallbackQuestions[0])
}

func (s *Service) handleInterviewAnswer(ctx context.Context, msg *tgbotapi.Message, session *Session, answer string) {
	if session.CurrentQuestionOrder <= 0 || session.CurrentQuestionOrder > len(fallbackQuestions) {
		s.log.Warn("invalid question order", "telegram_id", session.TelegramID, "order", session.CurrentQuestionOrder)
		s.sendText(msg.Chat.ID, "Что-то сбилось в интервью. Давай начнём заново через /start.")
		session.State = StateIdle
		return
	}

	question := fallbackQuestions[session.CurrentQuestionOrder-1]
	if len(session.Questions) >= session.CurrentQuestionOrder {
		question = session.Questions[session.CurrentQuestionOrder-1]
	}

	qa := llm.QA{
		Order:    session.CurrentQuestionOrder,
		Question: question,
		Answer:   answer,
	}
	session.Answers = append(session.Answers, qa)

	if err := s.metrics.AddAnswer(ctx, session.InterviewID, metricsclient.AddAnswerRequest{
		QuestionOrder: qa.Order,
		QuestionText:  qa.Question,
		AnswerText:    qa.Answer,
	}); err != nil {
		s.log.Warn("failed to save answer in metrics-service", "telegram_id", msg.From.ID, "error", err)
	}
	s.log.Info("interview answer recorded", "telegram_id", msg.From.ID, "interview_id", session.InterviewID, "question_order", qa.Order)
	s.trackEvent(ctx, msg.From.ID, session.InterviewID, "answer_saved", map[string]any{"question_order": qa.Order})

	if len(session.Answers) >= maxQuestions {
		s.generateResult(ctx, msg.Chat.ID, session)
		return
	}

	nextOrder := len(session.Answers) + 1
	nextQuestion := s.nextQuestion(ctx, *session, nextOrder)
	session.CurrentQuestionOrder = nextOrder
	s.askQuestion(msg.Chat.ID, session, nextQuestion)
}

func (s *Service) nextQuestion(ctx context.Context, session Session, order int) string {
	focus := requiredQuestionFocus[order-1]
	question, err := s.llm.GenerateNextQuestion(ctx, session.LLMContext(order, focus))
	if err != nil {
		s.log.Warn("failed to generate next question, using fallback", "telegram_id", session.TelegramID, "order", order, "error", err)
		return fallbackQuestions[order-1]
	}
	if strings.TrimSpace(question) == "" || questionAlreadyAsked(question, session.Answers) {
		return fallbackQuestions[order-1]
	}
	return question
}

func (s *Service) askQuestion(chatID int64, session *Session, question string) {
	order := session.CurrentQuestionOrder
	if len(session.Questions) == order-1 {
		session.Questions = append(session.Questions, question)
	} else if len(session.Questions) >= order {
		session.Questions[order-1] = question
	}
	s.sendText(chatID, fmt.Sprintf("Вопрос %d/%d\n\n%s", order, maxQuestions, question))
}

func (s *Service) generateResult(ctx context.Context, chatID int64, session *Session) {
	session.State = StateGenerating
	s.sendText(chatID, "Спасибо. Собираю описание для резюме: bullet points, summary, навыки и риски.")

	cleanAnswers := answeredOnly(session.Answers)
	session.Answers = cleanAnswers

	if err := s.metrics.CompleteInterview(ctx, session.InterviewID); err != nil {
		s.log.Warn("failed to complete interview in metrics-service", "telegram_id", session.TelegramID, "error", err)
	}

	result, err := s.llm.GenerateResumeDescription(ctx, session.LLMContext(maxQuestions, "итоговое описание"))
	if err != nil {
		s.log.Error("failed to generate result", "telegram_id", session.TelegramID, "error", err)
		s.trackEvent(ctx, session.TelegramID, session.InterviewID, "generation_failed", map[string]any{"error": err.Error()})
		session.State = StateIdle
		s.sendText(chatID, "Не смог сейчас вызвать LLM. Интервью не потерялось в чате, но результат лучше сгенерировать позже через /start.")
		return
	}

	created, err := s.metrics.CreateGeneration(ctx, metricsclient.GenerationRequest{
		InterviewID:    session.InterviewID,
		Bullets:        result.Bullets,
		ProjectSummary: result.ProjectSummary,
		Skills:         result.Skills,
		RiskWarnings:   combinedRiskWarnings(result),
		RawLLMResponse: result.RawResponse,
	})
	if err != nil {
		s.log.Warn("failed to save generation in metrics-service", "telegram_id", session.TelegramID, "error", err)
	} else {
		session.GenerationID = created.GenerationID
		s.log.Info("resume generation saved", "telegram_id", session.TelegramID, "interview_id", session.InterviewID, "generation_id", created.GenerationID)
	}
	s.trackEvent(ctx, session.TelegramID, session.InterviewID, "generation_created", map[string]any{"generation_id": session.GenerationID})

	s.sendLongText(chatID, formatGeneratedResult(result))
	session.State = StateWaitingReadyFeedback
	s.sendWithKeyboard(chatID, "Готов ли ты вставить это описание в своё реальное резюме без серьёзных правок?", readyKeyboard())
}

func (s *Service) handleReadyFeedback(userID int64, chatID int64, value string) {
	session := s.getSession(userID)
	if session == nil || session.State != StateWaitingReadyFeedback {
		s.sendText(chatID, "Сейчас этот ответ не ожидается. Начать заново можно через /start.")
		return
	}

	switch value {
	case "yes", "mostly":
		session.ReadyToUse = true
		session.NeedsMajorEdits = false
	case "no":
		session.ReadyToUse = false
		session.NeedsMajorEdits = true
	default:
		s.sendText(chatID, "Выбери один из вариантов.")
		return
	}
	session.State = StateWaitingIssueFeedback
	s.sendWithKeyboard(chatID, "Есть ли ощущение, что текст содержит воду, неточность или накрутку опыта?", issueKeyboard())
}

func (s *Service) handleIssueFeedback(userID int64, chatID int64, value string) {
	session := s.getSession(userID)
	if session == nil || session.State != StateWaitingIssueFeedback {
		s.sendText(chatID, "Сейчас этот ответ не ожидается.")
		return
	}

	if !validIssue(value) {
		s.sendText(chatID, "Выбери один из вариантов.")
		return
	}
	session.IssueType = value
	session.State = StateWaitingPayment
	s.sendWithKeyboard(chatID, "Был бы ты готов заплатить за улучшенную упаковку проекта или резюме целиком?", paymentKeyboard())
}

func (s *Service) handlePaymentFeedback(userID int64, chatID int64, value string) {
	session := s.getSession(userID)
	if session == nil || session.State != StateWaitingPayment {
		s.sendText(chatID, "Сейчас этот ответ не ожидается.")
		return
	}

	if !validPayment(value) {
		s.sendText(chatID, "Выбери один из вариантов.")
		return
	}
	session.PaymentWillingness = value
	session.State = StateWaitingTextFeedback
	s.sendText(chatID, "Что улучшить? Напиши свободным текстом. Можно коротко.")
}

func (s *Service) handleTextFeedback(ctx context.Context, msg *tgbotapi.Message, session *Session, comment string) {
	if err := s.metrics.CreateFeedback(ctx, metricsclient.FeedbackRequest{
		InterviewID:        session.InterviewID,
		GenerationID:       session.GenerationID,
		ReadyToUse:         session.ReadyToUse,
		NeedsMajorEdits:    session.NeedsMajorEdits,
		IssueType:          session.IssueType,
		PaymentWillingness: session.PaymentWillingness,
		Comment:            comment,
	}); err != nil {
		s.log.Warn("failed to save feedback in metrics-service", "telegram_id", msg.From.ID, "error", err)
	}

	s.trackEvent(ctx, msg.From.ID, session.InterviewID, "feedback_saved", map[string]any{
		"ready_to_use":        session.ReadyToUse,
		"issue_type":          session.IssueType,
		"payment_willingness": session.PaymentWillingness,
	})

	session.State = StateCompleted
	s.sendText(msg.Chat.ID, "Спасибо! Твой фидбек помогает понять, стоит ли развивать LinkedOut дальше.")
}

func (s *Service) handleStats(ctx context.Context, msg *tgbotapi.Message) {
	if !s.isAdmin(msg.From.ID) {
		s.sendText(msg.Chat.ID, "Эта команда доступна только админу.")
		return
	}

	s.log.Info("stats command requested", "telegram_id", msg.From.ID, "chat_id", msg.Chat.ID)
	summary, err := s.metrics.Summary(ctx)
	if err != nil {
		s.log.Warn("failed to load stats", "telegram_id", msg.From.ID, "error", err)
		s.sendText(msg.Chat.ID, "Не смог получить статистику из metrics-service.")
		return
	}

	s.log.Info("stats command completed", "telegram_id", msg.From.ID, "started_interviews", summary.StartedInterviews, "completed_interviews", summary.CompletedInterviews, "completion_rate", summary.CompletionRate, "mvp_success", summary.MVPSuccess)
	text := fmt.Sprintf(`Статистика LinkedOut

Начато интервью: %d
Завершено интервью: %d
Completion rate: %.0f%%
Среднее время интервью: %.1f мин
Ready-to-use: %.0f%% (%d)
Готовы платить: %d
Manual review pass rate: %.0f%% (%d/%d)
Not enough data: %t
MVP success: %t`,
		summary.StartedInterviews,
		summary.CompletedInterviews,
		summary.CompletionRate*100,
		summary.AvgInterviewDurationMinutes,
		summary.ReadyToUseRate*100,
		summary.ReadyToUseCount,
		summary.PaymentWillingUsers,
		summary.ManualReviewPassRate*100,
		summary.ManualReviewPassed,
		summary.ManualReviewTotal,
		summary.NotEnoughData,
		summary.MVPSuccess,
	)
	s.sendText(msg.Chat.ID, text)
}

func (s *Service) handlePendingReviews(ctx context.Context, msg *tgbotapi.Message) {
	if !s.isAdmin(msg.From.ID) {
		s.sendText(msg.Chat.ID, "Эта команда доступна только админу.")
		return
	}

	reviews, err := s.metrics.PendingReviews(ctx, 5)
	if err != nil {
		s.log.Warn("failed to load pending reviews", "telegram_id", msg.From.ID, "error", err)
		s.sendText(msg.Chat.ID, "Не смог получить список ручных проверок.")
		return
	}
	if len(reviews) == 0 {
		s.sendText(msg.Chat.ID, "Нет результатов, ожидающих ручной проверки.")
		return
	}

	var b strings.Builder
	b.WriteString("Последние результаты без ручной проверки:\n\n")
	for _, review := range reviews {
		fmt.Fprintf(&b, "ID: %s\nUser: %d\nRole: %s, project: %s\nBullets:\n%s\nSummary:\n%s\n\n",
			review.GenerationID,
			review.TelegramID,
			review.TargetRole,
			review.ProjectType,
			truncate(review.Bullets, 700),
			truncate(review.ProjectSummary, 400),
		)
	}
	b.WriteString("Проверка: /review <generation_id> <status> <comment>\nСтатусы: pass, fail_water, fail_inaccuracy, fail_exaggeration, fail_other")
	s.sendLongText(msg.Chat.ID, b.String())
}

func (s *Service) handleReview(ctx context.Context, msg *tgbotapi.Message) {
	if !s.isAdmin(msg.From.ID) {
		s.sendText(msg.Chat.ID, "Эта команда доступна только админу.")
		return
	}

	args := strings.TrimSpace(msg.CommandArguments())
	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 2 {
		s.sendText(msg.Chat.ID, "Формат: /review <generation_id> <status> <comment>")
		return
	}

	generationID := strings.TrimSpace(parts[0])
	status := strings.TrimSpace(parts[1])
	comment := ""
	if len(parts) == 3 {
		comment = strings.TrimSpace(parts[2])
	}
	if !validReviewStatus(status) {
		s.sendText(msg.Chat.ID, "Статус должен быть: pass, fail_water, fail_inaccuracy, fail_exaggeration, fail_other")
		return
	}

	if err := s.metrics.CreateManualReview(ctx, metricsclient.ManualReviewRequest{
		GenerationID:       generationID,
		ReviewerTelegramID: msg.From.ID,
		Status:             status,
		Comment:            comment,
	}); err != nil {
		s.log.Warn("failed to save manual review", "telegram_id", msg.From.ID, "error", err)
		s.sendText(msg.Chat.ID, "Не смог сохранить ручную проверку.")
		return
	}

	s.sendText(msg.Chat.ID, "Ручная проверка сохранена.")
}

func (s *Service) getSession(userID int64) *Session {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	return s.sessions[userID]
}

func (s *Service) setSession(userID int64, session *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[userID] = session
}

func (s *Service) cancelSession(userID int64) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, userID)
}

func (s *Service) isAdmin(userID int64) bool {
	_, ok := s.adminIDs[userID]
	return ok
}

func (s *Service) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := s.api.Send(msg); err != nil {
		s.log.Warn("send telegram message failed", "chat_id", chatID, "error", err)
	}
}

func (s *Service) sendLongText(chatID int64, text string) {
	for _, chunk := range splitTelegramText(text, 3900) {
		s.sendText(chatID, chunk)
	}
}

func (s *Service) sendWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	if _, err := s.api.Send(msg); err != nil {
		s.log.Warn("send telegram message failed", "chat_id", chatID, "error", err)
	}
}

func (s *Service) trackEvent(ctx context.Context, telegramID int64, interviewID string, eventType string, payload any) {
	request := metricsclient.EventRequest{
		TelegramID:  &telegramID,
		InterviewID: interviewID,
		EventType:   eventType,
		Payload:     payload,
	}
	if err := s.metrics.CreateEvent(ctx, request); err != nil {
		s.log.Warn("failed to send event to metrics-service", "telegram_id", telegramID, "event_type", eventType, "error", err)
	}
}

func answeredOnly(answers []llm.QA) []llm.QA {
	result := make([]llm.QA, 0, len(answers))
	for _, answer := range answers {
		if strings.TrimSpace(answer.Answer) != "" {
			result = append(result, answer)
		}
	}
	return result
}

func questionAlreadyAsked(question string, answers []llm.QA) bool {
	normalized := normalize(question)
	for _, answer := range answers {
		if normalize(answer.Question) == normalized {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, "?.! ")
	return value
}

func validIssue(value string) bool {
	switch value {
	case "none", "water", "inaccuracy", "exaggeration", "other":
		return true
	default:
		return false
	}
}

func validPayment(value string) bool {
	switch value {
	case "one_project", "full_resume", "maybe", "none":
		return true
	default:
		return false
	}
}

func validReviewStatus(value string) bool {
	switch value {
	case "pass", "fail_water", "fail_inaccuracy", "fail_exaggeration", "fail_other":
		return true
	default:
		return false
	}
}

func formatGeneratedResult(result llm.GeneratedResult) string {
	return fmt.Sprintf(`Готово. Черновик для резюме:

Bullet points:
%s

Краткое описание проекта:
%s

Подтверждённые навыки:
%s

Что звучит рискованно:
%s

Что лучше не писать, если не сможешь защитить на интервью:
%s`,
		emptyDash(result.Bullets),
		emptyDash(result.ProjectSummary),
		emptyDash(result.Skills),
		emptyDash(result.RiskWarnings),
		emptyDash(result.DontWrite),
	)
}

func combinedRiskWarnings(result llm.GeneratedResult) string {
	risk := strings.TrimSpace(result.RiskWarnings)
	dontWrite := strings.TrimSpace(result.DontWrite)
	if dontWrite == "" {
		return risk
	}
	if risk == "" {
		return "Что лучше не писать:\n" + dontWrite
	}
	return risk + "\n\nЧто лучше не писать:\n" + dontWrite
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func truncate(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func splitTelegramText(value string, limit int) []string {
	runes := []rune(value)
	if len(runes) <= limit {
		return []string{value}
	}

	chunks := make([]string, 0, int(math.Ceil(float64(len(runes))/float64(limit))))
	for len(runes) > 0 {
		end := limit
		if len(runes) < end {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[:end]))
		runes = runes[end:]
	}
	return chunks
}
