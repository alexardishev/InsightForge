package validate

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Validate[T any](t T) (bool, error) {
	validate := validator.New()
	err := validate.Struct(t)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			fmt.Printf("Поле %s не прошло валидацию: %s\n", err.Field(), err.Tag())
			return false, status.Error(codes.InvalidArgument, "плохой аргумент")
		}

	}
	return true, nil
}
