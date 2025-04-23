package propfind

import (
	"errors"

	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/samber/mo"
)

type ResponseMap map[string]mo.Result[props.Property]

type RequestType int

const (
	RequestTypeProp     RequestType = iota // Propfind request
	RequestTypePropName                    // Propname request (only return property names)
	RequestTypeAllProp                     // Allprop request (return all properties)
)

var (
	ErrNotFound   = errors.New("HTTP 404: Property not found")
	ErrForbidden  = errors.New("HTTP 403: Forbidden access to the resource")
	ErrInternal   = errors.New("HTTP 500: Internal server error")
	ErrBadRequest = errors.New("HTTP 400: Bad request")
)
