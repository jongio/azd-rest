package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// renderXML renders an XML response body with two-space indentation.
func renderXML(body []byte) (string, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("xml format requires a valid XML response: empty body")
	}

	dec := xml.NewDecoder(bytes.NewReader(trimmed))
	var b strings.Builder
	enc := xml.NewEncoder(&b)
	enc.Indent("", "  ")

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("xml format requires a valid XML response: %w", err)
		}

		if charData, ok := tok.(xml.CharData); ok && len(bytes.TrimSpace(charData)) == 0 {
			continue
		}

		if proc, ok := tok.(xml.ProcInst); ok && strings.EqualFold(proc.Target, "xml") {
			writeXMLDeclaration(&b, proc)
			continue
		}

		if err := enc.EncodeToken(tok); err != nil {
			return "", fmt.Errorf("failed to encode response as xml: %w", err)
		}
	}

	if err := enc.Flush(); err != nil {
		return "", fmt.Errorf("failed to encode response as xml: %w", err)
	}

	out := b.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

func writeXMLDeclaration(b *strings.Builder, proc xml.ProcInst) {
	b.WriteString("<?")
	b.WriteString(proc.Target)
	if len(proc.Inst) > 0 {
		b.WriteByte(' ')
		b.Write(proc.Inst)
	}
	b.WriteString("?>\n")
}
