package main

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func printStructuredOutput(format string, payload interface{}) error {
	switch format {
	case "json":
		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("print structured output: %w", err)
		}
		_, err = fmt.Println(string(body))
		return err
	case "yaml":
		body, err := yaml.Marshal(payload)
		if err != nil {
			return fmt.Errorf("print structured output: %w", err)
		}
		_, err = fmt.Print(string(body))
		return err
	default:
		_, err := fmt.Printf("%+v\n", payload)
		return err
	}
}
