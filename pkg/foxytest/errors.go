package foxytest

import "errors"

var (
	ErrFailedToParseInput  = errors.New("failed to parse input")
	ErrFailedToParseOutput = errors.New("failed to parse output")
)
