package errors

import "errors"

var (
	ErrorLinkAlreadyExists   = errors.New("error link is already exists")
	ErrorLinkTooShort        = errors.New("error link is too short")
	ErrorInvalidSymbolInLink = errors.New("error invalid symbol in link")
	ErrorLinkNotFound        = errors.New("error link not found")
)
