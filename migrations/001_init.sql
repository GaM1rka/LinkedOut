CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id BIGINT UNIQUE NOT NULL,
    username TEXT,
    first_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS interviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_role TEXT NOT NULL,
    project_type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('started', 'in_progress', 'completed', 'abandoned')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ NULL,
    duration_seconds INT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS interview_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    question_order INT NOT NULL,
    question_text TEXT NOT NULL,
    answer_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    bullets TEXT NOT NULL,
    project_summary TEXT NOT NULL,
    skills TEXT NOT NULL,
    risk_warnings TEXT NOT NULL,
    raw_llm_response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS feedbacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    generation_id UUID NOT NULL REFERENCES generations(id) ON DELETE CASCADE,
    ready_to_use BOOLEAN NOT NULL,
    needs_major_edits BOOLEAN NOT NULL,
    issue_type TEXT NULL CHECK (issue_type IS NULL OR issue_type IN ('none', 'water', 'inaccuracy', 'exaggeration', 'other')),
    payment_willingness TEXT NOT NULL CHECK (payment_willingness IN ('none', 'maybe', 'one_project', 'full_resume')),
    comment TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS manual_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id UUID NOT NULL REFERENCES generations(id) ON DELETE CASCADE,
    reviewer_telegram_id BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pass', 'fail_water', 'fail_inaccuracy', 'fail_exaggeration', 'fail_other')),
    comment TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    interview_id UUID NULL REFERENCES interviews(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    payload JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_interviews_user_id ON interviews(user_id);
CREATE INDEX IF NOT EXISTS idx_interview_answers_interview_id ON interview_answers(interview_id);
CREATE INDEX IF NOT EXISTS idx_generations_interview_id ON generations(interview_id);
CREATE INDEX IF NOT EXISTS idx_feedbacks_interview_id ON feedbacks(interview_id);
CREATE INDEX IF NOT EXISTS idx_manual_reviews_generation_id ON manual_reviews(generation_id);
CREATE INDEX IF NOT EXISTS idx_events_interview_id ON events(interview_id);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
