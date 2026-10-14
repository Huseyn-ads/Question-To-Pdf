-- +goose Up
CREATE TABLE quiz_answer_options(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  position INTEGER NOT NULL CHECK (position > 0),
  is_correct BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE(quiz_question_id, position)

);

-- +goose Down
DROP TABLE quiz_answer_options;
