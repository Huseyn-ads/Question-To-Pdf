-- +goose Up
CREATE TABLE quizzes(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  question_bank_id UUID NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
  question_count INTEGER NOT NULL CHECK (question_count > 0),
  time_limit_minutes INTEGER NOT NULL DEFAULT 0 CHECK (time_limit_minutes BETWEEN 0 AND 180),
  shuffle_options BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quizzes_user_id ON quizzes(user_id);

-- +goose Down
DROP TABLE quizzes;
