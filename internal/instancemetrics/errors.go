package instancemetrics

import "errors"

// ErrNoPayload is returned when no metrics blob exists for the instance (e.g. Redis miss).
var ErrNoPayload = errors.New("no instance metrics payload available")
