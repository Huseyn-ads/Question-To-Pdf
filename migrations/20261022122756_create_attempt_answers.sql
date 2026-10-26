
-- +goose Up

CREATE TABLE attempt_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,

    quiz_question_id UUID NOT NULL REFERENCES quiz_questions(id),

    quiz_answer_option_id UUID NOT NULL REFERENCES quiz_answer_options(id),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (quiz_attempt_id, quiz_question_id)
);

CREATE INDEX idx_attempt_answers_quiz_question_id ON attempt_answers(quiz_question_id);


-- +goose Down

DROP TABLE attempt_answers;
