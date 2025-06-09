package validate

import (
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Validate[T any](t T) (bool, error) {
	validate := validator.New()
	err := validate.Struct(t)
	if err != nil {
		for range err.(validator.ValidationErrors) {
			return false, status.Error(codes.InvalidArgument, "плохой аргумент")
		}

	}
	return true, nil
}
