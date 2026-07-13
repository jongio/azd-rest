package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// templateConfigError signals that the --template value could not be loaded or
// parsed (a bad @file path or invalid template syntax). It reports exit code 2
// (invalid configuration) through the ExitCoder contract so main exits before,
// or independent of, any response handling.
type templateConfigError struct{ msg string }

func (e *templateConfigError) Error() string { return e.msg }

// ExitCode returns 2 for an invalid --template value.
func (e *templateConfigError) ExitCode() int { return 2 }

// templateFuncs are the helper functions exposed to --template templates. They
// cover the common shaping needs (embedding JSON, changing case, joining a
// list) without pulling in a large templating dependency.
var templateFuncs = template.FuncMap{
	"json":  templateJSON,
	"upper": func(v any) string { return strings.ToUpper(fmt.Sprint(v)) },
	"lower": func(v any) string { return strings.ToLower(fmt.Sprint(v)) },
	"join":  templateJoin,
}

// templateJSON marshals a value to compact JSON so a template can embed a
// nested object or array as a string.
func templateJSON(v any) (string, error) {
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// templateJoin concatenates the elements of a list with a separator. It accepts
// the []any that JSON arrays decode into as well as a plain []string, and
// renders each element with fmt.Sprint so numbers and strings both work.
func templateJoin(sep string, list any) string {
	switch v := list.(type) {
	case []string:
		return strings.Join(v, sep)
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprint(item)
		}
		return strings.Join(parts, sep)
	default:
		return fmt.Sprint(list)
	}
}

// resolveTemplateText returns the template source for a --template value. A
// value that starts with "@" is treated as a path and the file is read; any
// other value is used as the template text directly. A missing file is returned
// as a templateConfigError so the process exits with code 2.
func resolveTemplateText(value string) (string, error) {
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}
	path := value[1:]
	data, err := os.ReadFile(path) // #nosec G304 -- User-specified template file path via --template @file is intentional.
	if err != nil {
		return "", &templateConfigError{msg: fmt.Sprintf("could not read --template file %q: %v", path, err)}
	}
	return string(data), nil
}

// parseTemplate resolves and compiles a --template value. Parse and file errors
// are returned as templateConfigError (exit 2) so an invalid template fails fast
// before any request is made.
func parseTemplate(value string) (*template.Template, error) {
	text, err := resolveTemplateText(value)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("response").Funcs(templateFuncs).Parse(text)
	if err != nil {
		return nil, &templateConfigError{msg: fmt.Sprintf("invalid --template syntax: %v", err)}
	}
	return tmpl, nil
}

// renderTemplate parses the --template value, decodes the JSON body, and
// executes the template against it. A non-JSON body is reported as a clear
// error. Template execution errors are wrapped with context.
func renderTemplate(value string, body []byte) (string, error) {
	tmpl, err := parseTemplate(value)
	if err != nil {
		return "", err
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return "", fmt.Errorf("--template needs a JSON response, but the body did not parse as JSON: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("--template failed to render: %w", err)
	}
	return out.String(), nil
}
