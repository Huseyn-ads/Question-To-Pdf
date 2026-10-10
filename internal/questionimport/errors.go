package questionimport

import "errors"

var (
	ErrParsingForm      = errors.New("error parsing form")
	ErrFileNameRequired = errors.New("file name is required")
	ErrFileUpload       = errors.New("error while uploading file")
	ErrFileCreate       = errors.New("error while creating file")
)
