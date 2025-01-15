package api

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

func init() {
	enLocale := en.New()
	uni = ut.New(enLocale, enLocale)
	Validate = validator.New()
	Trans, _ = uni.GetTranslator("en")

	if err := en_lang.RegisterDefaultTranslations(Validate, Trans); err != nil {
		panic("Failed to register default translations: " + err.Error())
	}
}

func TranslateError(err error) []string {
	errs := err.(validator.ValidationErrors)
	translatedErrors := make([]string, 0, len(errs))

	for _, e := range errs {
		translatedErrors = append(translatedErrors, e.Translate(Trans))
	}

	return translatedErrors
}
