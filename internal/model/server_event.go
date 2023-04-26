package model

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

// ServerEvent is a flat struct encapsulating a parsed log event
type ServerEvent struct {
	Server Server
	*logparse.Results
}
