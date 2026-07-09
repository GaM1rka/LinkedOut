package metrics

import "testing"

func TestCalculateSummaryAvgInterviewDuration(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           2,
		CompletedInterviews:         2,
		AvgInterviewDurationSeconds: 780,
		FeedbackTotal:               1,
		ReadyToUseCount:             1,
		PaymentWillingUsers:         3,
		ManualReviewTotal:           1,
		ManualReviewPassed:          1,
	})

	if summary.AvgInterviewDurationSeconds != 780 {
		t.Fatalf("expected avg duration seconds 780, got %v", summary.AvgInterviewDurationSeconds)
	}
	if summary.AvgInterviewDurationMinutes != 13 {
		t.Fatalf("expected avg duration minutes 13, got %v", summary.AvgInterviewDurationMinutes)
	}
}

func TestCalculateSummaryReadyToUseRate(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           10,
		CompletedInterviews:         10,
		AvgInterviewDurationSeconds: 600,
		FeedbackTotal:               10,
		ReadyToUseCount:             7,
		PaymentWillingUsers:         3,
		ManualReviewTotal:           10,
		ManualReviewPassed:          7,
	})

	if summary.ReadyToUseRate != 0.7 {
		t.Fatalf("expected ready_to_use_rate 0.7, got %v", summary.ReadyToUseRate)
	}
}

func TestCalculateSummaryPaymentWillingUsers(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           10,
		CompletedInterviews:         10,
		AvgInterviewDurationSeconds: 600,
		FeedbackTotal:               10,
		ReadyToUseCount:             7,
		PaymentWillingUsers:         4,
		ManualReviewTotal:           10,
		ManualReviewPassed:          7,
	})

	if summary.PaymentWillingUsers != 4 {
		t.Fatalf("expected 4 payment willing users, got %d", summary.PaymentWillingUsers)
	}
}

func TestCalculateSummaryManualReviewPassRate(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           10,
		CompletedInterviews:         10,
		AvgInterviewDurationSeconds: 600,
		FeedbackTotal:               10,
		ReadyToUseCount:             7,
		PaymentWillingUsers:         3,
		ManualReviewTotal:           10,
		ManualReviewPassed:          8,
	})

	if summary.ManualReviewPassRate != 0.8 {
		t.Fatalf("expected manual_review_pass_rate 0.8, got %v", summary.ManualReviewPassRate)
	}
}

func TestCalculateSummaryMVPSuccessTrue(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           10,
		CompletedInterviews:         10,
		AvgInterviewDurationSeconds: 780,
		FeedbackTotal:               10,
		ReadyToUseCount:             7,
		PaymentWillingUsers:         3,
		ManualReviewTotal:           10,
		ManualReviewPassed:          7,
	})

	if !summary.MVPSuccess {
		t.Fatalf("expected mvp_success true")
	}
	if summary.NotEnoughData {
		t.Fatalf("expected not_enough_data false")
	}
}

func TestCalculateSummaryMVPSuccessFalseWhenTargetsMissed(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           10,
		CompletedInterviews:         10,
		AvgInterviewDurationSeconds: 960,
		FeedbackTotal:               10,
		ReadyToUseCount:             6,
		PaymentWillingUsers:         2,
		ManualReviewTotal:           10,
		ManualReviewPassed:          6,
	})

	if summary.MVPSuccess {
		t.Fatalf("expected mvp_success false")
	}
}

func TestCalculateSummaryNotEnoughData(t *testing.T) {
	summary := CalculateSummary(RawStats{
		StartedInterviews:           1,
		CompletedInterviews:         1,
		AvgInterviewDurationSeconds: 600,
	})

	if !summary.NotEnoughData {
		t.Fatalf("expected not_enough_data true")
	}
	if summary.MVPSuccess {
		t.Fatalf("expected mvp_success false when data is not enough")
	}
}
