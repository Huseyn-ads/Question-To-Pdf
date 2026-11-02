package attempt

import (
	"context"
	"errors"
	"fmt"
	"qtp/internal/quiz"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttemptRepository struct {
	db *pgxpool.Pool
}

func NewAttemptRepository(db *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{
		db: db,
	}
}

func (repo *AttemptRepository) Start(ctx context.Context, userID string, quizID string) (*Attempt, bool, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin start attempt transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Locking the quiz serializes concurrent starts for the same quiz. The
	// partial unique index on active attempts remains the final database-level
	// protection against duplicate in-progress attempts.
	const findQuizQuery = `
		SELECT time_limit_minutes
		FROM quizzes
		WHERE id = $1 AND user_id = $2
		FOR UPDATE;
	`

	var timeLimitMinutes int
	if err := tx.QueryRow(ctx, findQuizQuery, quizID, userID).Scan(&timeLimitMinutes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, quiz.ErrQuizNotFound
		}

		return nil, false, fmt.Errorf("find owned quiz for attempt: %w", err)
	}

	const findActiveAttemptQuery = `
		SELECT
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count,
			deadline_at IS NOT NULL AND deadline_at <= NOW()
		FROM quiz_attempts
		WHERE quiz_id = $1
			AND user_id = $2
			AND status = 'in_progress'
		FOR UPDATE;
	`

	activeAttempt := &Attempt{}
	var deadlineExpired bool

	err = tx.QueryRow(ctx, findActiveAttemptQuery, quizID, userID).Scan(
		&activeAttempt.ID,
		&activeAttempt.QuizID,
		&activeAttempt.UserID,
		&activeAttempt.Status,
		&activeAttempt.StartedAt,
		&activeAttempt.DeadlineAt,
		&activeAttempt.FinishedAt,
		&activeAttempt.CorrectCount,
		&activeAttempt.GradableCount,
		&deadlineExpired,
	)

	switch {
	case err == nil && !deadlineExpired:
		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf("commit existing attempt transaction: %w", err)
		}

		return activeAttempt, false, nil

	case err == nil && deadlineExpired:
		const expireAttemptQuery = `
			UPDATE quiz_attempts
			SET
				status = 'expired',
				finished_at = NOW()
			WHERE id = $1 AND status = 'in_progress';
		`

		result, err := tx.Exec(ctx, expireAttemptQuery, activeAttempt.ID)
		if err != nil {
			return nil, false, fmt.Errorf("expire active attempt: %w", err)
		}

		if result.RowsAffected() != 1 {
			return nil, false, fmt.Errorf("expire active attempt: expected one updated row, got %d", result.RowsAffected())
		}

	case errors.Is(err, pgx.ErrNoRows):
		// There is no active attempt, so a new one can be created.

	default:
		return nil, false, fmt.Errorf("find active attempt: %w", err)
	}

	const createAttemptQuery = `
		INSERT INTO quiz_attempts (
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at
		)
		VALUES (
			$1,
			$2,
			'in_progress',
			NOW(),
			CASE
				WHEN $3 > 0 THEN NOW() + ($3 * INTERVAL '1 minute')
				ELSE NULL
			END
		)
		RETURNING
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count;
	`

	createdAttempt := &Attempt{}
	err = tx.QueryRow(
		ctx,
		createAttemptQuery,
		quizID,
		userID,
		timeLimitMinutes,
	).Scan(
		&createdAttempt.ID,
		&createdAttempt.QuizID,
		&createdAttempt.UserID,
		&createdAttempt.Status,
		&createdAttempt.StartedAt,
		&createdAttempt.DeadlineAt,
		&createdAttempt.FinishedAt,
		&createdAttempt.CorrectCount,
		&createdAttempt.GradableCount,
	)
	if err != nil {
		return nil, false, fmt.Errorf("create attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit start attempt transaction: %w", err)
	}

	return createdAttempt, true, nil
}

func (repo *AttemptRepository) GetAttempt(ctx context.Context, userID string, attemptID string) (*AttemptDetails, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin get attempt transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	details := &AttemptDetails{
		Questions: make([]AttemptQuestion, 0),
		Answers:   make([]AttemptAnswer, 0),
	}

	const findAttemptQuery = `
		SELECT
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count,
			NOW(),
			status = 'in_progress'
				AND deadline_at IS NOT NULL
				AND deadline_at <= NOW()
		FROM quiz_attempts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE;
	`

	var deadlineExpired bool
	err = tx.QueryRow(ctx, findAttemptQuery, attemptID, userID).Scan(
		&details.ID,
		&details.QuizID,
		&details.UserID,
		&details.Status,
		&details.StartedAt,
		&details.DeadlineAt,
		&details.FinishedAt,
		&details.CorrectCount,
		&details.GradableCount,
		&details.ServerTime,
		&deadlineExpired,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("find owned attempt: %w", err)
	}

	if deadlineExpired {
		const expireAttemptQuery = `
			UPDATE quiz_attempts
			SET
				status = 'expired',
				finished_at = NOW()
			WHERE id = $1 AND status = 'in_progress'
			RETURNING status, finished_at;
		`

		if err := tx.QueryRow(ctx, expireAttemptQuery, details.ID).Scan(
			&details.Status,
			&details.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("expire attempt while getting it: %w", err)
		}
	}

	const findQuestionsQuery = `
		SELECT
			qq.id,
			qq.quiz_id,
			qq.text,
			qq.position,
			qao.id,
			qao.quiz_question_id,
			qao.text,
			qao.position
		FROM quiz_questions qq
		JOIN quiz_answer_options qao
			ON qao.quiz_question_id = qq.id
		WHERE qq.quiz_id = $1
		ORDER BY qq.position, qao.position;
	`

	rows, err := tx.Query(ctx, findQuestionsQuery, details.QuizID)
	if err != nil {
		return nil, fmt.Errorf("query attempt questions: %w", err)
	}

	for rows.Next() {
		question := AttemptQuestion{}
		option := AttemptOption{}

		if err := rows.Scan(
			&question.ID,
			&question.QuizID,
			&question.Text,
			&question.Position,
			&option.ID,
			&option.QuizQuestionID,
			&option.Text,
			&option.Position,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan attempt question: %w", err)
		}

		isFirstQuestion := len(details.Questions) == 0
		if isFirstQuestion || details.Questions[len(details.Questions)-1].ID != question.ID {
			question.Options = make([]AttemptOption, 0)
			details.Questions = append(details.Questions, question)
		}

		lastQuestionIndex := len(details.Questions) - 1
		details.Questions[lastQuestionIndex].Options = append(
			details.Questions[lastQuestionIndex].Options,
			option,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate attempt questions: %w", err)
	}
	rows.Close()

	const findAnswersQuery = `
		SELECT
			id,
			quiz_attempt_id,
			quiz_question_id,
			quiz_answer_option_id,
			updated_at
		FROM attempt_answers
		WHERE quiz_attempt_id = $1
		ORDER BY updated_at, id;
	`

	answerRows, err := tx.Query(ctx, findAnswersQuery, details.ID)
	if err != nil {
		return nil, fmt.Errorf("query attempt answers: %w", err)
	}

	for answerRows.Next() {
		answer := AttemptAnswer{}
		if err := answerRows.Scan(
			&answer.ID,
			&answer.QuizAttemptID,
			&answer.QuizQuestionID,
			&answer.QuizAnswerOptionID,
			&answer.UpdatedAt,
		); err != nil {
			answerRows.Close()
			return nil, fmt.Errorf("scan attempt answer: %w", err)
		}

		details.Answers = append(details.Answers, answer)
	}

	if err := answerRows.Err(); err != nil {
		answerRows.Close()
		return nil, fmt.Errorf("iterate attempt answers: %w", err)
	}
	answerRows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get attempt transaction: %w", err)
	}

	return details, nil
}

func (repo *AttemptRepository) SaveAnswer(ctx context.Context, userID string, attemptID string, quizQuestionID string, optionID string) (*AttemptAnswer, error) {

	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin save answer transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var quizID string
	var status QuizStatus
	var deadlineAt *time.Time
	var deadlineExpired bool
	const findAttemptQuery = `
	SELECT qa.quiz_id,
	qa.status,
	qa.deadline_at,
	qa.deadline_at IS NOT NULL AND qa.deadline_at <= NOW() AS deadline_expired
	FROM quiz_attempts qa
	JOIN quizzes q
	ON q.id = qa.quiz_id
	AND q.user_id = qa.user_id
	WHERE qa.id = $1 AND qa.user_id = $2
	FOR UPDATE OF qa;
	`

	err = tx.QueryRow(ctx, findAttemptQuery, attemptID, userID).Scan(&quizID, &status, &deadlineAt, &deadlineExpired)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("find attempt query: %w", err)
	}

	if status != "in_progress" || deadlineExpired {
		return nil, ErrAttemptClosed
	}

	var validOption bool

	const checkOptionQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM quiz_questions qq
		JOIN quiz_answer_options qao
		ON qao.quiz_question_id = qq.id
		WHERE qq.id = $1
		AND qq.quiz_id = $2
		AND qao.id = $3
	);
	`

	err = tx.QueryRow(ctx, checkOptionQuery, quizQuestionID, quizID, optionID).Scan(&validOption)

	if err != nil {
		return nil, fmt.Errorf("check quiz question and option: %w", err)
	}

	if !validOption {
		return nil, ErrAnswerTargetNotFound
	}

	attemptAnswer := AttemptAnswer{}

	const saveAnswerQuery = `
		INSERT INTO attempt_answers (
		quiz_attempt_id,
		quiz_question_id,
		quiz_answer_option_id
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (quiz_attempt_id, quiz_question_id)
		DO UPDATE SET
		quiz_answer_option_id = EXCLUDED.quiz_answer_option_id,
		updated_at = NOW()
		RETURNING
		id,
		quiz_attempt_id,
		quiz_question_id,
		quiz_answer_option_id,
		updated_at;
	`

	err = tx.QueryRow(ctx, saveAnswerQuery, attemptID, quizQuestionID, optionID).
		Scan(
			&attemptAnswer.ID,
			&attemptAnswer.QuizAttemptID,
			&attemptAnswer.QuizQuestionID,
			&attemptAnswer.QuizAnswerOptionID,
			&attemptAnswer.UpdatedAt,
		)

	if err != nil {
		return nil, fmt.Errorf("save answer: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit save answer transaction: %w", err)
	}

	return &attemptAnswer, nil
}

func (repo *AttemptRepository) Finish(
	ctx context.Context,
	userID string,
	attemptID string,
) (*AttemptResult, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin finish transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const findAttemptQuery = `
		SELECT
			qa.quiz_id,
			qa.status,
			qa.started_at,
			qa.deadline_at IS NOT NULL
				AND qa.deadline_at <= NOW()
				AS deadline_expired
		FROM quiz_attempts qa
		JOIN quizzes q
			ON q.id = qa.quiz_id
			AND q.user_id = qa.user_id
		WHERE qa.id = $1
			AND qa.user_id = $2
		FOR UPDATE OF qa;
	`

	var quizID string
	var status QuizStatus
	var startedAt time.Time
	var deadlineExpired bool

	err = tx.QueryRow(
		ctx,
		findAttemptQuery,
		attemptID,
		userID,
	).Scan(
		&quizID,
		&status,
		&startedAt,
		&deadlineExpired,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("find attempt for finish: %w", err)
	}

	// Если попытка ещё активна, определяем причину завершения.
	// Уже закрытая попытка сохраняет свой старый статус.
	finalStatus := status

	if status == StatusInProgress {
		finalStatus = StatusCompleted

		if deadlineExpired {
			finalStatus = StatusExpired
		}
	}

	const resultQuery = `
		SELECT
			qq.id,
			qq.quiz_id,
			qq.text,
			qq.position,

			aa.quiz_answer_option_id,

			qao.id,
			qao.quiz_question_id,
			qao.text,
			qao.position,
			qao.is_correct

		FROM quiz_questions qq

		LEFT JOIN attempt_answers aa
			ON aa.quiz_attempt_id = $1
			AND aa.quiz_question_id = qq.id

		LEFT JOIN quiz_answer_options qao
			ON qao.quiz_question_id = qq.id

		WHERE qq.quiz_id = $2

		ORDER BY qq.position, qao.position;
	`

	rows, err := tx.Query(
		ctx,
		resultQuery,
		attemptID,
		quizID,
	)
	if err != nil {
		return nil, fmt.Errorf("query finish result: %w", err)
	}

	questions := make([]ResultQuestion, 0)

	for rows.Next() {
		var question ResultQuestion
		var selectedOptionID *string

		// Эти значения могут быть NULL из-за LEFT JOIN.
		var optionID *string
		var optionQuestionID *string
		var optionText *string
		var optionPosition *int
		var optionIsCorrect *bool

		err := rows.Scan(
			&question.ID,
			&question.QuizID,
			&question.Text,
			&question.Position,

			&selectedOptionID,

			&optionID,
			&optionQuestionID,
			&optionText,
			&optionPosition,
			&optionIsCorrect,
		)

		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan finish result: %w", err)
		}

		// SQL возвращает вопрос отдельно для каждого его варианта.
		// Поэтому сам вопрос добавляем только один раз.
		isFirstQuestion := len(questions) == 0

		isNewQuestion := isFirstQuestion ||
			questions[len(questions)-1].ID != question.ID

		if isNewQuestion {
			question.SelectedOptionID = selectedOptionID
			question.Options = make([]ResultOption, 0)

			questions = append(questions, question)
		}

		// У вопроса нет вариантов.
		if optionID == nil {
			continue
		}

		if optionQuestionID == nil ||
			optionText == nil ||
			optionPosition == nil ||
			optionIsCorrect == nil {
			rows.Close()

			return nil, fmt.Errorf(
				"incomplete result option data: option_id=%s",
				*optionID,
			)
		}

		// Проверяем, выбрал ли пользователь текущий вариант.
		isSelected := selectedOptionID != nil &&
			*selectedOptionID == *optionID

		option := ResultOption{
			ID:             *optionID,
			QuizQuestionID: *optionQuestionID,
			Text:           *optionText,
			Position:       *optionPosition,
			IsCorrect:      *optionIsCorrect,
			IsSelected:     isSelected,
		}

		lastQuestionIndex := len(questions) - 1

		questions[lastQuestionIndex].Options = append(
			questions[lastQuestionIndex].Options,
			option,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate finish result: %w", err)
	}

	rows.Close()

	attemptResult := &AttemptResult{
		AttemptID:               attemptID,
		QuizID:                  quizID,
		Status:                  finalStatus,
		TotalQuestions:          len(questions),
		StartedAt:               startedAt,
		CorrectAnswersList:      make([]ResultQuestion, 0),
		WrongAnswersList:        make([]ResultQuestion, 0),
		UnansweredQuestionsList: make([]ResultQuestion, 0),
	}

	// Теперь каждый вопрос содержит все свои варианты.
	// Распределяем вопросы по трём спискам.
	for _, question := range questions {
		// Пользователь не выбрал ни один вариант.
		if question.SelectedOptionID == nil {
			attemptResult.UnansweredQuestions++

			attemptResult.UnansweredQuestionsList = append(
				attemptResult.UnansweredQuestionsList,
				question,
			)

			continue
		}

		selectedOptionFound := false
		selectedOptionIsCorrect := false

		// Находим выбранный пользователем вариант.
		for _, option := range question.Options {
			if option.IsSelected {
				selectedOptionFound = true
				selectedOptionIsCorrect = option.IsCorrect
				break
			}
		}

		if !selectedOptionFound {
			return nil, fmt.Errorf(
				"selected option does not belong to question: question_id=%s option_id=%s",
				question.ID,
				*question.SelectedOptionID,
			)
		}

		question.IsCorrect = &selectedOptionIsCorrect

		attemptResult.AnsweredQuestions++

		if selectedOptionIsCorrect {
			attemptResult.CorrectAnswers++

			attemptResult.CorrectAnswersList = append(
				attemptResult.CorrectAnswersList,
				question,
			)

			continue
		}

		attemptResult.WrongAnswers++

		attemptResult.WrongAnswersList = append(
			attemptResult.WrongAnswersList,
			question,
		)
	}

	if attemptResult.TotalQuestions > 0 {
		attemptResult.Percentage =
			float64(attemptResult.CorrectAnswers) /
				float64(attemptResult.TotalQuestions) * 100
	}

	const finishAttemptQuery = `
		UPDATE quiz_attempts
		SET
			status = $2,
			finished_at = COALESCE(finished_at, NOW()),
			correct_count = $3,
			gradable_count = $4
		WHERE id = $1
			AND user_id = $5
		RETURNING finished_at;
	`

	err = tx.QueryRow(
		ctx,
		finishAttemptQuery,
		attemptID,
		finalStatus,
		attemptResult.CorrectAnswers,
		attemptResult.TotalQuestions,
		userID,
	).Scan(
		&attemptResult.FinishedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("finish attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit finish transaction: %w", err)
	}

	return attemptResult, nil
}

func (repo *AttemptRepository) GetResult(
	ctx context.Context,
	userID string,
	attemptID string,
) (*AttemptResult, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin get result transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const findAttemptQuery = `
		SELECT
			qa.quiz_id,
			qa.status,
			qa.started_at,
			qa.deadline_at,
			qa.finished_at,
			qa.deadline_at IS NOT NULL
				AND qa.deadline_at <= NOW()
				AS deadline_expired
		FROM quiz_attempts qa
		JOIN quizzes q
			ON q.id = qa.quiz_id
			AND q.user_id = qa.user_id
		WHERE qa.id = $1
			AND qa.user_id = $2
		FOR UPDATE OF qa;
	`

	var quizID string
	var status QuizStatus
	var startedAt time.Time
	var deadlineAt *time.Time
	var finishedAt *time.Time
	var deadlineExpired bool

	err = tx.QueryRow(
		ctx,
		findAttemptQuery,
		attemptID,
		userID,
	).Scan(
		&quizID,
		&status,
		&startedAt,
		&deadlineAt,
		&finishedAt,
		&deadlineExpired,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("find attempt for result: %w", err)
	}

	// Активная попытка, время которой ещё не закончилось.
	// Если deadlineAt == nil, deadlineExpired тоже будет false.
	if status == StatusInProgress && !deadlineExpired {
		return nil, ErrAttemptInProgress
	}

	finalStatus := status
	shouldFinalize := false

	// Попытка ещё записана как in_progress,
	// но её дедлайн уже наступил.
	if status == StatusInProgress && deadlineExpired {
		finalStatus = StatusExpired
		shouldFinalize = true
	}

	const resultQuery = `
		SELECT
			qq.id,
			qq.quiz_id,
			qq.text,
			qq.position,

			aa.quiz_answer_option_id,

			qao.id,
			qao.quiz_question_id,
			qao.text,
			qao.position,
			qao.is_correct

		FROM quiz_questions qq

		LEFT JOIN attempt_answers aa
			ON aa.quiz_attempt_id = $1
			AND aa.quiz_question_id = qq.id

		LEFT JOIN quiz_answer_options qao
			ON qao.quiz_question_id = qq.id

		WHERE qq.quiz_id = $2

		ORDER BY qq.position, qao.position;
	`

	rows, err := tx.Query(
		ctx,
		resultQuery,
		attemptID,
		quizID,
	)
	if err != nil {
		return nil, fmt.Errorf("query attempt result: %w", err)
	}

	questions := make([]ResultQuestion, 0)

	for rows.Next() {
		var question ResultQuestion
		var selectedOptionID *string

		var optionID *string
		var optionQuestionID *string
		var optionText *string
		var optionPosition *int
		var optionIsCorrect *bool

		err := rows.Scan(
			&question.ID,
			&question.QuizID,
			&question.Text,
			&question.Position,

			&selectedOptionID,

			&optionID,
			&optionQuestionID,
			&optionText,
			&optionPosition,
			&optionIsCorrect,
		)

		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan attempt result: %w", err)
		}

		isFirstQuestion := len(questions) == 0

		isNewQuestion := isFirstQuestion ||
			questions[len(questions)-1].ID != question.ID

		if isNewQuestion {
			question.SelectedOptionID = selectedOptionID
			question.Options = make([]ResultOption, 0)

			questions = append(questions, question)
		}

		if optionID == nil {
			continue
		}

		if optionQuestionID == nil ||
			optionText == nil ||
			optionPosition == nil ||
			optionIsCorrect == nil {
			rows.Close()

			return nil, fmt.Errorf(
				"incomplete result option data: option_id=%s",
				*optionID,
			)
		}

		isSelected := selectedOptionID != nil &&
			*selectedOptionID == *optionID

		option := ResultOption{
			ID:             *optionID,
			QuizQuestionID: *optionQuestionID,
			Text:           *optionText,
			Position:       *optionPosition,
			IsCorrect:      *optionIsCorrect,
			IsSelected:     isSelected,
		}

		lastQuestionIndex := len(questions) - 1

		questions[lastQuestionIndex].Options = append(
			questions[lastQuestionIndex].Options,
			option,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate attempt result: %w", err)
	}

	rows.Close()

	attemptResult := &AttemptResult{
		AttemptID:               attemptID,
		QuizID:                  quizID,
		Status:                  finalStatus,
		TotalQuestions:          len(questions),
		StartedAt:               startedAt,
		CorrectAnswersList:      make([]ResultQuestion, 0),
		WrongAnswersList:        make([]ResultQuestion, 0),
		UnansweredQuestionsList: make([]ResultQuestion, 0),
	}

	for _, question := range questions {
		if question.SelectedOptionID == nil {
			attemptResult.UnansweredQuestions++

			attemptResult.UnansweredQuestionsList = append(
				attemptResult.UnansweredQuestionsList,
				question,
			)

			continue
		}

		selectedOptionFound := false
		selectedOptionIsCorrect := false

		for _, option := range question.Options {
			if option.IsSelected {
				selectedOptionFound = true
				selectedOptionIsCorrect = option.IsCorrect
				break
			}
		}

		if !selectedOptionFound {
			return nil, fmt.Errorf(
				"selected option does not belong to question: question_id=%s option_id=%s",
				question.ID,
				*question.SelectedOptionID,
			)
		}

		question.IsCorrect = &selectedOptionIsCorrect
		attemptResult.AnsweredQuestions++

		if selectedOptionIsCorrect {
			attemptResult.CorrectAnswers++

			attemptResult.CorrectAnswersList = append(
				attemptResult.CorrectAnswersList,
				question,
			)

			continue
		}

		attemptResult.WrongAnswers++

		attemptResult.WrongAnswersList = append(
			attemptResult.WrongAnswersList,
			question,
		)
	}

	if attemptResult.TotalQuestions > 0 {
		attemptResult.Percentage =
			float64(attemptResult.CorrectAnswers) /
				float64(attemptResult.TotalQuestions) * 100
	}

	if shouldFinalize {
		const expireAttemptQuery = `
			UPDATE quiz_attempts
			SET
				status = 'expired',
				finished_at = deadline_at,
				correct_count = $2,
				gradable_count = $3
			WHERE id = $1
				AND user_id = $4
				AND status = 'in_progress'
			RETURNING finished_at;
		`

		err = tx.QueryRow(
			ctx,
			expireAttemptQuery,
			attemptID,
			attemptResult.CorrectAnswers,
			attemptResult.TotalQuestions,
			userID,
		).Scan(
			&attemptResult.FinishedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrAttemptNotFound
			}

			return nil, fmt.Errorf(
				"expire attempt while getting result: %w",
				err,
			)
		}
	} else {
		if finishedAt == nil {
			return nil, fmt.Errorf(
				"closed attempt has no finished_at: attempt_id=%s",
				attemptID,
			)
		}

		attemptResult.FinishedAt = *finishedAt
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"commit get result transaction: %w",
			err,
		)
	}

	return attemptResult, nil
}

func (repo *AttemptRepository) GetHistory(ctx context.Context, userID string, quizID string, limit int, offset int) ([]AttemptSummary, error) {
	attemptSummaries := make([]AttemptSummary, 0)

	const checkQuery = `
	SELECT id
	FROM quizzes
	WHERE id = $1 AND user_id = $2;
	`

	var dummyID int
	err := repo.db.QueryRow(ctx, checkQuery, quizID, userID).Scan(&dummyID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, quiz.ErrQuizNotFound
		}
		return nil, fmt.Errorf("check quiz query: %w", err)
	}

	var expiredAttemptID string
	const findExpiredAttemptQuery = `
		SELECT qa.id
		FROM quiz_attempts qa
		JOIN quizzes q
			ON q.id = qa.quiz_id
			AND q.user_id = qa.user_id
		WHERE qa.quiz_id = $1
			AND qa.user_id = $2
			AND qa.status = 'in_progress'
			AND qa.deadline_at IS NOT NULL
			AND qa.deadline_at <= NOW();
	`

	err = repo.db.QueryRow(ctx, findExpiredAttemptQuery, quizID, userID).Scan(&expiredAttemptID)

	switch {
	case err == nil:
		_, err = repo.GetResult(
			ctx,
			userID,
			expiredAttemptID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"finalize expired attempt: %w",
				err,
			)
		}

	case errors.Is(err, pgx.ErrNoRows):
		// Просроченной активной попытки нет.
		// Просто продолжаем получать историю.

	default:
		return nil, fmt.Errorf(
			"find expired attempt: %w",
			err,
		)
	}

	const getAttemptsQuery = `
		SELECT
			qa.id,
			qa.quiz_id,
			qa.status,
			qa.started_at,
			qa.deadline_at,
			qa.finished_at,
			qa.correct_count,
			qa.gradable_count,
			CASE
				WHEN qa.status IN ('completed', 'expired') THEN
					CASE
						WHEN qa.gradable_count > 0 THEN
							qa.correct_count * 100.0 / qa.gradable_count
						ELSE 0
					END
				ELSE NULL
			END AS percentage
		FROM quiz_attempts qa
		JOIN quizzes q
			ON q.id = qa.quiz_id
			AND q.user_id = qa.user_id
		WHERE qa.quiz_id = $1
			AND qa.user_id = $2
		ORDER BY qa.started_at DESC, qa.id DESC
		LIMIT $3 OFFSET $4;
	`

	rows, err := repo.db.Query(ctx, getAttemptsQuery, quizID, userID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("get attempts: %w", err)
	}
	for rows.Next() {
		attemptSummary := AttemptSummary{}
		var percentage *float64

		err = rows.Scan(
			&attemptSummary.AttemptID,
			&attemptSummary.QuizID,
			&attemptSummary.Status,
			&attemptSummary.StartedAt,
			&attemptSummary.DeadlineAt,
			&attemptSummary.FinishedAt,
			&attemptSummary.CorrectAnswers,
			&attemptSummary.GradableCount,
			&percentage,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan attempt history: %w", err)
		}
		attemptSummary.Percentage = percentage

		attemptSummaries = append(attemptSummaries, attemptSummary)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate attempt history: %w", err)
	}

	rows.Close()

	return attemptSummaries, nil
}
