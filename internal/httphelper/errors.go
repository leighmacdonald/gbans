package httphelper

type Code int

const (
	EntityDoesNotExist Code = iota + 1
	ParamInvalid
)
