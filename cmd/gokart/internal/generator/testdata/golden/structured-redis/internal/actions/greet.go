package actions

import (
	"errors"
	"strings"
)

var ErrNameRequired = errors.New("name is required")

// GreetInput holds the input parameters for the Greet action.
type GreetInput struct {
	Name string
	Loud bool
}

// Validate checks that the input is valid.
func (i GreetInput) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return ErrNameRequired
	}
	return nil
}

// Greet returns a greeting message for the given input.
func Greet(input GreetInput) (string, error) {
	if err := input.Validate(); err != nil {
		return "", err
	}
	msg := "Hello, " + input.Name
	if input.Loud {
		msg = strings.ToUpper(msg) + "!"
	}
	return msg, nil
}
