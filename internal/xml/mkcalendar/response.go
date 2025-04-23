package mkcalendar

import (
	"errors"
	"reflect"
	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/props"
)

// ParseRequest parses a MKCALENDAR XML request and returns a map of property
// names to decoded Property values. Unknown props are skipped.
func ParseRequest(xmlStr string) (map[string]props.Property, error) {
	result := make(map[string]props.Property)

	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return result, err
	}

	mk := doc.FindElement("//mkcalendar")
	if mk == nil {
		// no top‐level mkcalendar → still no properties (could be error if you prefer)
		return result, errors.New("invalid MKCALENDAR request: missing mkcalendar element")
	}

	set := mk.FindElement("set")
	if set == nil {
		// per tests, missing <set> just yields empty map, no error
		return result, nil
	}

	prop := set.FindElement("prop")
	if prop == nil {
		// per tests, missing <prop> just yields empty map, no error
		return result, nil
	}

	for _, e := range prop.ChildElements() {
		// strip prefix, lowercase
		local := e.Tag
		if strings.Contains(local, ":") {
			parts := strings.Split(local, ":")
			local = parts[len(parts)-1]
		}
		local = strings.ToLower(local)

		if proto, ok := props.PropNameToStruct[local]; ok {
			t := reflect.TypeOf(proto).Elem()
			inst := reflect.New(t).Interface().(props.Property)
			if err := inst.Decode(e); err == nil {
				result[local] = inst
			}
		}
	}

	return result, nil
}
