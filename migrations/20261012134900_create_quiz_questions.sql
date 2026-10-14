-- +goose Up
CREATE TABLE quiz_questions(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  position INTEGER NOT NULL CHECK (position > 0),
  UNIQUE (quiz_id, position)
);

-- +goose Down
DROP TABLE quiz_questions;
