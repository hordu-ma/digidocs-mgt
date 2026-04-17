package service

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrInvalidTransition = errors.New("invalid state transition")
var ErrUnauthorized = errors.New("unauthorized")
var ErrForbidden = errors.New("forbidden")
var ErrValidation = errors.New("validation error")
var ErrConflict = errors.New("resource already exists")
