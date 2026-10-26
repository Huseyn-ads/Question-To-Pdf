-- +goose Up
CREATE TABLE quiz_attempts(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_id UUID NOT NULL REFERENCES quizzes(id),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'expired')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deadline_at TIMESTAMPTZ NULL,
  finished_at TIMESTAMPTZ NULL,
  correct_count INTEGER NOT NULL DEFAULT 0 CHECK (correct_count >= 0),
  gradable_count INTEGER NOT NULL DEFAULT 0 CHECK (gradable_count >= 0)
);


CREATE INDEX idx_quiz_attempts_user_id
    ON quiz_attempts(user_id);

CREATE UNIQUE INDEX idx_quiz_attempts_active
    ON quiz_attempts(quiz_id, user_id)
    WHERE status = 'in_progress';


-- +goose Down
DROP TABLE quiz_attempts;
