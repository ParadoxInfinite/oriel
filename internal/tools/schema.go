package tools

import "fmt"

// Schema is a deliberately tiny JSON-Schema subset — enough to validate the
// flat, typed argument maps our tools take, without pulling in a dependency.
type Schema struct {
	Required []string        `json:"required,omitempty"`
	Props    map[string]Prop `json:"properties,omitempty"`
}

type Prop struct {
	Type        string   `json:"type"`           // "string" | "number" | "boolean"
	Enum        []string `json:"enum,omitempty"` // allowed values (strings only)
	Description string   `json:"description,omitempty"`
}

// Validate checks required presence, types, and enum membership.
func (s Schema) Validate(args map[string]any) error {
	for _, key := range s.Required {
		if _, ok := args[key]; !ok {
			return fmt.Errorf("missing required argument %q", key)
		}
	}
	for key, val := range args {
		prop, known := s.Props[key]
		if !known {
			return fmt.Errorf("unknown argument %q", key)
		}
		if err := checkType(key, prop, val); err != nil {
			return err
		}
	}
	return nil
}

func checkType(key string, prop Prop, val any) error {
	switch prop.Type {
	case "string":
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("argument %q must be a string", key)
		}
		if len(prop.Enum) > 0 && !contains(prop.Enum, s) {
			return fmt.Errorf("argument %q must be one of %v", key, prop.Enum)
		}
	case "number":
		// JSON numbers decode to float64.
		if _, ok := val.(float64); !ok {
			return fmt.Errorf("argument %q must be a number", key)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("argument %q must be a boolean", key)
		}
	default:
		return fmt.Errorf("argument %q has unsupported type %q", key, prop.Type)
	}
	return nil
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
