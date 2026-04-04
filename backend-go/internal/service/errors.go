package service

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrInvalidTransition = errors.New("invalid state transition")
