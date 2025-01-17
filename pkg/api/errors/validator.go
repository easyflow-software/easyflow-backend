package errors

import (
	"github.com/go-playground/locales/en"
	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_lang "github.com/go-playground/validator/v10/translations/en"
)

var (
	uni      *ut.UniversalTranslator
	Validate *validator.Validate
	Trans    ut.Translator
)

// Initializes the validator and translator with English locale.
// It sets up the validator instance and registers default English translations.
// Panics if registration of translations fails.
func init() {
	enLocale := en.New()
	uni = ut.New(enLocale, enLocale)
	Validate = validator.New()
	Trans, _ = uni.GetTranslator("en")
	if err := en_lang.RegisterDefaultTranslations(Validate, Trans); err != nil {
		panic("Failed to register default translations: " + err.Error())
	}
}

// TranslateError converts validation errors into human readable messages.
func TranslateError(err error) []string {
	errs := err.(validator.ValidationErrors)
	translatedErrors := make([]string, 0, len(errs))
	for _, e := range errs {
		translatedErrors = append(translatedErrors, e.Translate(Trans))
	}
	return translatedErrors
}
