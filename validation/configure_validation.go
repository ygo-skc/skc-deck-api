package validation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	V                 = validator.New()
	Translator        ut.Translator
	deckListNameRegex = regexp.MustCompile(`^[a-zA-Z0-9 !\-@]{3,}$`)
	cardIDRegex       = regexp.MustCompile(`^[0-9]{8}$`)
)

const (
	deckListNameValidator = "decklistname"
	deckMascotsValidator  = "deckmascots"
	requiredValidator     = "required"
	base64Validator       = "base64"
	urlValidator          = "url"
)

func init() {
	enTranslator := en.New()
	uni := ut.New(enTranslator, enTranslator)

	var found bool
	Translator, found = uni.GetTranslator("en")
	if !found {
		slog.Error("Translator not found", "locale", "en")
		os.Exit(1)
	}

	configureTranslations()
	configureCustomValidators()
}

type validationError struct {
	Field string `json:"field"`
	Hint  string `json:"hint"`
}

type ValidationErrors struct {
	Errors      []validationError `json:"errors"`
	TotalErrors int               `json:"totalErrors"`
}

func (e *ValidationErrors) HandleServerResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(res).Encode(e)
}

func handleValidationErrors(err validator.ValidationErrors) *ValidationErrors {
	validationErrors := []validationError{}
	for _, e := range err {
		validationErrors = append(validationErrors, validationError{Field: e.Field(), Hint: e.Translate(Translator)})
	}

	ve := ValidationErrors{Errors: validationErrors, TotalErrors: len(validationErrors)}
	slog.Error("Errors occurred while validating input", "total_errors", ve.TotalErrors, "errors", ve.Errors)
	return &ve
}
