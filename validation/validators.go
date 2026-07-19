package validation

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	cModel "github.com/ygo-skc/skc-go/common/v2/model"
)

// Add custom validators to handle validation scenarios not supported out of the box.
func configureCustomValidators() {
	V.RegisterValidation(deckListNameValidator, func(fl validator.FieldLevel) bool {
		return len(deckListNameRegex.FindAllString(fl.Field().String(), -1)) > 0
	})

	V.RegisterValidation(deckMascotsValidator, func(fl validator.FieldLevel) bool {
		mascots := fl.Field().Interface().(cModel.CardIDs)

		for ind, mascot := range mascots {
			if ind == 3 { // size constraint fails
				slog.Error("Deck Mascot array failed size constraint", "mascot_count", len(mascots))
				return false
			} else if len(cardIDRegex.FindAllString(mascot, -1)) == 0 { // regex constraint
				slog.Error("Deck Mascot ID not in proper format", "mascot_id", mascot)
				return false
			}
		}

		return true
	})
}
