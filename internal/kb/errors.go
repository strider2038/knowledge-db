package kb

import "github.com/muonsoft/errors"

var (
	ErrNodeNotFound   = errors.New("node not found")
	ErrInvalidPath    = errors.New("invalid path")
	ErrInvalidStructure = errors.New("invalid base structure")
)
