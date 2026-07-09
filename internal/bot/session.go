package bot

import (
	"time"

	"linkedout/internal/llm"
)

type State string

const (
	StateIdle                 State = "idle"
	StateChoosingRole         State = "choosing_role"
	StateChoosingProjectType  State = "choosing_project_type"
	StateInterviewing         State = "interviewing"
	StateGenerating           State = "generating"
	StateWaitingReadyFeedback State = "waiting_ready_to_use_feedback"
	StateWaitingIssueFeedback State = "waiting_issue_feedback"
	StateWaitingPayment       State = "waiting_payment_feedback"
	StateWaitingTextFeedback  State = "waiting_text_feedback"
	StateCompleted            State = "completed"
)

const maxQuestions = 7

var requiredQuestionFocus = []string{
	"название проекта и проблема, которую он решал",
	"личная роль пользователя в проекте",
	"что конкретно пользователь сделал сам",
	"какие технологии использовал",
	"самая сложная техническая часть",
	"измеримый или качественный результат",
	"что пользователь реально сможет объяснить на собеседовании",
}

var fallbackQuestions = []string{
	"Как называется проект и какую проблему он решал?",
	"Какая была твоя личная роль в проекте?",
	"Что конкретно ты сделал сам?",
	"Какие технологии использовал?",
	"Какая была самая сложная техническая часть?",
	"Был ли измеримый или качественный результат?",
	"Что из этого ты реально сможешь объяснить на собеседовании?",
}

type Session struct {
	TelegramID           int64
	MetricsUserID        string
	InterviewID          string
	GenerationID         string
	TargetRole           string
	ProjectType          string
	CurrentQuestionOrder int
	Questions            []string
	Answers              []llm.QA
	StartedAt            time.Time
	State                State

	ReadyToUse         bool
	NeedsMajorEdits    bool
	IssueType          string
	PaymentWillingness string
}

func (s Session) LLMContext(nextOrder int, requiredFocus string) llm.InterviewContext {
	return llm.InterviewContext{
		TargetRole:        s.TargetRole,
		ProjectType:       s.ProjectType,
		NextQuestionOrder: nextOrder,
		MaxQuestions:      maxQuestions,
		RequiredFocus:     requiredFocus,
		Answers:           s.Answers,
	}
}
