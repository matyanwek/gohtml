package gohtml

import (
	"errors"
)

// Errors.  These may serve as warnings, as well.
var (
	EmptyInputErr    = errors.New("empty input")
	EofErr           = errors.New("unexpected EOF")
	EntityErr        = errors.New("invalid entity")
	TokenErr         = errors.New("unexpected token")
	UnclosedTagErr   = errors.New("unclosed tag")
	EmptyContentErr  = errors.New("empty token content")
	EmptyTagStackErr = errors.New("empty tag stack")
	TagMismatchErr   = errors.New("mismatched tags")
)
