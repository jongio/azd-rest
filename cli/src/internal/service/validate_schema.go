package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaResourceName is the in-memory identifier the compiler uses for the
// user-supplied schema. It never touches the filesystem or network.
const schemaResourceName = "schema.json"

// validateSchemaUsageError signals invalid --validate-schema usage: a missing
// or unreadable schema file, a schema that is not valid JSON or not a valid
// JSON Schema, or a non-JSON response. It reports exit code 2 so main can tell
// it apart from a conformance failure, which is a plain error (exit 1).
type validateSchemaUsageError struct{ msg string }

func (e *validateSchemaUsageError) Error() string { return e.msg }

// ExitCode returns 2 for invalid --validate-schema usage.
func (e *validateSchemaUsageError) ExitCode() int { return 2 }

// validateResponseSchema validates the JSON response body against the JSON
// Schema in schemaPath. It returns nil when the response conforms. When the
// response does not conform it writes each validation error to errOut and
// returns a plain error so the command exits non-zero. A missing or invalid
// schema file, or a non-JSON response, returns a validateSchemaUsageError
// (exit 2).
func validateResponseSchema(errOut io.Writer, body []byte, schemaPath string) error {
	schemaRaw, err := os.ReadFile(schemaPath) // #nosec G304 -- User-specified schema path via --validate-schema flag is intentional.
	if err != nil {
		return &validateSchemaUsageError{msg: fmt.Sprintf("failed to read --validate-schema file %q: %v", schemaPath, err)}
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaRaw))
	if err != nil {
		return &validateSchemaUsageError{msg: fmt.Sprintf("--validate-schema file %q is not valid JSON: %v", schemaPath, err)}
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaResourceName, schemaDoc); err != nil {
		return &validateSchemaUsageError{msg: fmt.Sprintf("invalid JSON Schema in %q: %v", schemaPath, err)}
	}
	schema, err := compiler.Compile(schemaResourceName)
	if err != nil {
		return &validateSchemaUsageError{msg: fmt.Sprintf("invalid JSON Schema in %q: %v", schemaPath, err)}
	}

	if !client.IsJSON(body) {
		return &validateSchemaUsageError{msg: "--validate-schema requires a JSON response"}
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return &validateSchemaUsageError{msg: fmt.Sprintf("failed to parse JSON response for --validate-schema: %v", err)}
	}

	err = schema.Validate(instance)
	if err == nil {
		return nil
	}

	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return &validateSchemaUsageError{msg: fmt.Sprintf("--validate-schema failed: %v", err)}
	}

	messages := flattenSchemaErrors(ve.BasicOutput())
	for _, m := range messages {
		fmt.Fprintln(errOut, m)
	}

	noun := "errors"
	if len(messages) == 1 {
		noun = "error"
	}
	return fmt.Errorf("response does not conform to JSON Schema %q (%d %s)", schemaPath, len(messages), noun)
}

// flattenSchemaErrors walks the basic-output tree and returns one line per leaf
// failure in the form "instanceLocation: message", so callers can print a flat
// list of concrete problems instead of a nested structure.
func flattenSchemaErrors(unit *jsonschema.OutputUnit) []string {
	var messages []string
	var walk func(u *jsonschema.OutputUnit)
	walk = func(u *jsonschema.OutputUnit) {
		if u == nil {
			return
		}
		if u.Error != nil {
			loc := u.InstanceLocation
			if loc == "" {
				loc = "/"
			}
			messages = append(messages, fmt.Sprintf("%s: %s", loc, u.Error.String()))
		}
		for i := range u.Errors {
			walk(&u.Errors[i])
		}
	}
	walk(unit)

	if len(messages) == 0 {
		// The tree carried no leaf error text (unusual); fall back to a single
		// generic line so the user still sees a failure was reported.
		messages = append(messages, "response does not conform to the schema")
	}
	return messages
}
