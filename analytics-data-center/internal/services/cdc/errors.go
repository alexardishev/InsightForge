package cdc

import "errors"

// permanentError помечает ошибку, которая не исчезнет при повторной обработке
// того же сообщения (битый JSON, удалённая вьюха и т.п.).
type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent помечает ошибку как постоянную: listener не будет повторять
// обработку, а отправит сообщение в DLQ и закоммитит offset.
// Все остальные ошибки считаются временными и повторяются без ограничения.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// IsPermanent сообщает, помечена ли ошибка (или любая ошибка в её цепочке) как постоянная.
func IsPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}
