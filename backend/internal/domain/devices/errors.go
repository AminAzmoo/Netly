package devices

import "errors"

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrInvalidDevice  = errors.New("invalid device data")
)
