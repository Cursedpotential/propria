// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// paramsSchema is the JSON Schema subset the repair tools declare: an object
// of scalar properties with bounds, enums and a closed property set. The
// Workbench renders the same document; the validator enforces it here so a
// plan can never carry a knob a tool does not understand.
type paramsSchema struct {
	Type                 string                   `json:"type"`
	Description          string                   `json:"description,omitempty"`
	Properties           map[string]*paramsSchema `json:"properties,omitempty"`
	Required             []string                 `json:"required,omitempty"`
	AdditionalProperties *bool                    `json:"additionalProperties,omitempty"`
	Minimum              *float64                 `json:"minimum,omitempty"`
	Maximum              *float64                 `json:"maximum,omitempty"`
	Enum                 []any                    `json:"enum,omitempty"`
	Default              any                      `json:"default,omitempty"`
}

func parseParamsSchema(raw json.RawMessage) (*paramsSchema, error) {
	var schema paramsSchema
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&schema); err != nil {
		return nil, fmt.Errorf("params schema is not the supported JSON Schema subset: %w", err)
	}
	if schema.Type != "object" {
		return nil, errors.New("params schema must describe an object")
	}
	return &schema, nil
}

// validateParams checks params (absent or null means {}) against schema.
func validateParams(schemaRaw json.RawMessage, params json.RawMessage) error {
	schema, err := parseParamsSchema(schemaRaw)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(params)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		trimmed = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("params are not valid JSON: %w", err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return errors.New("params must be a JSON object")
	}
	var problems []string
	for _, name := range schema.Required {
		if _, present := object[name]; !present {
			problems = append(problems, fmt.Sprintf("%q is required", name))
		}
	}
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		property, known := schema.Properties[name]
		if !known {
			if schema.AdditionalProperties == nil || !*schema.AdditionalProperties {
				problems = append(problems, fmt.Sprintf("%q is not a parameter of this tool", name))
			}
			continue
		}
		if err := validateScalar(property, object[name]); err != nil {
			problems = append(problems, fmt.Sprintf("%q %v", name, err))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validateScalar(schema *paramsSchema, value any) error {
	switch schema.Type {
	case "integer", "number":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("must be a %s", schema.Type)
		}
		parsed, err := number.Float64()
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return fmt.Errorf("must be a %s", schema.Type)
		}
		if schema.Type == "integer" && parsed != math.Trunc(parsed) {
			return errors.New("must be a whole number")
		}
		if schema.Minimum != nil && parsed < *schema.Minimum {
			return fmt.Errorf("must be at least %v", *schema.Minimum)
		}
		if schema.Maximum != nil && parsed > *schema.Maximum {
			return fmt.Errorf("must be at most %v", *schema.Maximum)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("must be true or false")
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return errors.New("must be a string")
		}
		if len(schema.Enum) > 0 {
			for _, allowed := range schema.Enum {
				if allowed == text {
					return nil
				}
			}
			return fmt.Errorf("must be one of %v", schema.Enum)
		}
	default:
		return fmt.Errorf("has unsupported schema type %q", schema.Type)
	}
	return nil
}

// canonicalParams renders params as compact JSON with sorted keys, so two
// steps that ask for the same thing compare equal.
func canonicalParams(params json.RawMessage) string {
	trimmed := bytes.TrimSpace(params)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "{}"
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return string(trimmed)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return string(trimmed)
	}
	return string(encoded)
}
