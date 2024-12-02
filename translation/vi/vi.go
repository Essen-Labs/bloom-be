package vietnamese

import (
	"errors"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

const (
	locale = "vi"
)

// Init initializes the english locale translations
func Init(uni *ut.UniversalTranslator, validate *validator.Validate) error {
	vi, found := uni.GetTranslator(locale)
	if !found {
		return errors.New("Translation not found")
	}

	err := vi.Add("username", "tên đăng nhập", false)
	if err != nil {
		return err
	}
	err = vi.Add("password", "mật khẩu", false)
	if err != nil {
		return err
	}
	err = vi.Add("amount", "số lượng sản phẩm", false)
	if err != nil {
		return err
	}
	err = vi.Add("bad request", "không thể thực hiện yêu cầu", false)
	if err != nil {
		return err
	}

	// validator translations & Overrides
	err = RegisterDefaultTranslations(validate, vi)
	if err != nil {
		return errors.New("Error adding default translations: " + err.Error())
	}

	if err := vi.VerifyTranslations(); err != nil {
		return errors.New("Missing Translations: " + err.Error())
	}
	return nil
}
