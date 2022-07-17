package error

import "fmt"

var _ error = Error{}

type Error struct {
	wrapped error
}

func (e Error) Error() string {
	return fmt.Sprintf(`%v`, e.wrapped)
}

func (e Error) String() string {
	return fmt.Sprintf(`%v`, e.wrapped)
}

func New(wrapped error) (e Error) {
	return Error{
		wrapped: wrapped,
	}
}
