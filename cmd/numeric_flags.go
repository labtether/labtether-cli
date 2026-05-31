package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

type boundedIntValue struct {
	value int
	name  string
	unit  string
	min   int
	max   int
}

func newBoundedIntValue(defaultValue int, name string, unit string, min int, max int) *boundedIntValue {
	return &boundedIntValue{
		value: defaultValue,
		name:  name,
		unit:  unit,
		min:   min,
		max:   max,
	}
}

func (v *boundedIntValue) Set(raw string) error {
	value, ok := strictDecimalInt(raw)
	if !ok || value < v.min || value > v.max {
		if v.unit != "" {
			return fmt.Errorf("%s must be between %d and %d %s", v.name, v.min, v.max, v.unit)
		}
		return fmt.Errorf("%s must be between %d and %d", v.name, v.min, v.max)
	}
	v.value = value
	return nil
}

func (v *boundedIntValue) String() string {
	return strconv.Itoa(v.value)
}

func (v *boundedIntValue) Type() string {
	return "int"
}

func strictDecimalInt(raw string) (int, bool) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return 0, false
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}
