package metrics

type RawStats struct {
	StartedInterviews           int
	CompletedInterviews         int
	AvgInterviewDurationSeconds float64
	FeedbackTotal               int
	ReadyToUseCount             int
	PaymentWillingUsers         int
	ManualReviewTotal           int
	ManualReviewPassed          int
}

type Targets struct {
	MaxAvgDurationMinutes  float64 `json:"max_avg_duration_minutes"`
	MinReadyToUseRate      float64 `json:"min_ready_to_use_rate"`
	MinPaymentWillingUsers int     `json:"min_payment_willing_users"`
	MinManualReviewRate    float64 `json:"min_manual_review_pass_rate"`
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
	Targets                     Targets `json:"targets"`
}

func CalculateSummary(raw RawStats) Summary {
	targets := Targets{
		MaxAvgDurationMinutes:  15,
		MinReadyToUseRate:      0.7,
		MinPaymentWillingUsers: 3,
		MinManualReviewRate:    0.7,
	}

	summary := Summary{
		StartedInterviews:           raw.StartedInterviews,
		CompletedInterviews:         raw.CompletedInterviews,
		AvgInterviewDurationSeconds: raw.AvgInterviewDurationSeconds,
		AvgInterviewDurationMinutes: raw.AvgInterviewDurationSeconds / 60,
		ReadyToUseCount:             raw.ReadyToUseCount,
		PaymentWillingUsers:         raw.PaymentWillingUsers,
		ManualReviewTotal:           raw.ManualReviewTotal,
		ManualReviewPassed:          raw.ManualReviewPassed,
		Targets:                     targets,
	}

	if raw.StartedInterviews > 0 {
		summary.CompletionRate = float64(raw.CompletedInterviews) / float64(raw.StartedInterviews)
	}
	if raw.FeedbackTotal > 0 {
		summary.ReadyToUseRate = float64(raw.ReadyToUseCount) / float64(raw.FeedbackTotal)
	}
	if raw.ManualReviewTotal > 0 {
		summary.ManualReviewPassRate = float64(raw.ManualReviewPassed) / float64(raw.ManualReviewTotal)
	}

	summary.NotEnoughData = raw.CompletedInterviews == 0 || raw.FeedbackTotal == 0 || raw.ManualReviewTotal == 0
	if summary.NotEnoughData {
		return summary
	}

	summary.MVPSuccess = summary.AvgInterviewDurationMinutes <= targets.MaxAvgDurationMinutes &&
		summary.ReadyToUseRate >= targets.MinReadyToUseRate &&
		raw.PaymentWillingUsers >= targets.MinPaymentWillingUsers &&
		summary.ManualReviewPassRate >= targets.MinManualReviewRate

	return summary
}
