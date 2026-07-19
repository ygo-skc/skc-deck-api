package validation

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/ygo-skc/skc-deck-api/model"
)

// validate deck list
func Validate(dl model.DeckList) *ValidationErrors {
	if err := V.Struct(dl); err != nil {
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			slog.Error("Validator returned an unexpected error type", "err", err)
			return &ValidationErrors{
				Errors:      []validationError{{Field: "deckList", Hint: "Deck list could not be validated due to an internal error."}},
				TotalErrors: 1,
			}
		}
		return handleValidationErrors(validationErrs)
	} else {
		return nil
	}
}
