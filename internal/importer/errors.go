package importer

import "errors"

var (
	ErrMathNotSupported = errors.New(
		"math formulas are not supported",
	)

	ErrMediaNotSupported = errors.New(
		"images and embedded objects are not supported",
	)
)
