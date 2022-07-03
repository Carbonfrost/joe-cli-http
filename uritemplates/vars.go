package uritemplates

import (
	"fmt"
)

// Vars provides template variables
type Vars map[string]interface{}

func (t Vars) Add(v ...*Var) {
	for _, u := range v {
		switch val := u.Value.(type) {
		case map[string]interface{}:
			t.setMapHelper(u.Name, val)
		case []interface{}:
			t.setArrayHelper(u.Name, val)
		case string:
			t.setStringHelper(u.Name, val)
		default:
			panic("unreachable!")
		}
	}
}

func (t Vars) setStringHelper(name, value string) error {
	t[name] = value
	return nil
}

func (t Vars) setArrayHelper(name string, values []interface{}) error {
	if current, ok := t[name]; ok {
		switch c := current.(type) {
		case []interface{}:
			t[name] = append(c, values...)
			return nil
		case string:
			t[name] = append([]interface{}{c}, values...)
			return nil
		case map[string]interface{}:
			for _, v := range values {
				c[fmt.Sprint(v)] = ""
			}
			t[name] = c
			return nil
		}
	}

	t[name] = values
	return nil
}

func (t Vars) setMapHelper(name string, values map[string]interface{}) error {
	if current, ok := t[name]; ok {
		switch c := current.(type) {
		case []interface{}:
			return fmt.Errorf("existing value is array, cannot apply map")
		case string:
			return fmt.Errorf("existing value is array, cannot apply string")
		case map[string]interface{}:
			for k, v := range values {
				c[k] = v
			}
			t[name] = c
			return nil
		}
	}

	t[name] = values
	return nil
}
