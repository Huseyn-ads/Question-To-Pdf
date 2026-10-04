package questionbank

import "github.com/google/uuid"

func validateBankID(bankID string) error {
	if _, err := uuid.Parse(bankID); err != nil {
		return ErrInvalidQuestionBankID
	}

	return nil
}
