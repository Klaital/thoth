package goupc

import "errors"

// ErrNotFound is returned when go-upc.com has no record for the barcode.
var ErrNotFound = errors.New("barcode not found")
