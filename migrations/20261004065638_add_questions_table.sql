-- +goose Up
CREATE TABLE questions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  question_bank_id UUID NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_question_bank_id
ON questions(question_bank_id);

-- +goose Down
DROP TABLE questions;
