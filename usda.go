package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type tokenKind uint8

const (
	tokWord tokenKind = iota
	tokString
	tokPunct
	tokNewline
)

type token struct {
	kind tokenKind
	text string
	line int
}

func lexUSDA(src string) ([]token, error) {
	var out []token
	line := 1
	for i := 0; i < len(src); {
		ch := src[i]
		switch {
		case ch == '\n':
			out = append(out, token{kind: tokNewline, line: line})
			line++
			i++
		case ch == ' ' || ch == '\t' || ch == '\r':
			i++
		case ch == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case ch == '"':
			startLine := line
			i++
			var b strings.Builder
			closed := false
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					b.WriteByte(src[i+1])
					i += 2
					continue
				}
				if src[i] == '"' {
					i++
					closed = true
					break
				}
				if src[i] == '\n' {
					line++
				}
				b.WriteByte(src[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("line %d: unterminated string", startLine)
			}
			out = append(out, token{kind: tokString, text: b.String(), line: startLine})
		case strings.ContainsRune("{}()=,[];", rune(ch)):
			out = append(out, token{kind: tokPunct, text: string(ch), line: line})
			i++
		default:
			start := i
			for i < len(src) {
				c := src[i]
				if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '#' || c == '"' || strings.ContainsRune("{}()=,[];", rune(c)) {
					break
				}
				i++
			}
			if start == i {
				return nil, fmt.Errorf("line %d: unexpected character %q", line, src[i])
			}
			out = append(out, token{kind: tokWord, text: src[start:i], line: line})
		}
	}
	return out, nil
}

type usdaParser struct {
	toks  []token
	pos   int
	layer *Layer
}

func ParseUSDA(layerID, file string, src []byte) (*Layer, error) {
	toks, err := lexUSDA(string(src))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	p := &usdaParser{
		toks:  toks,
		layer: &Layer{ID: layerID, File: file},
	}
	p.skipSeparators()
	if p.at("(") {
		if err := p.parseLayerMetadata(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}
	for {
		p.skipSeparators()
		if p.eof() {
			break
		}
		t := p.peek()
		if t.kind == tokPunct && t.text == "}" {
			return nil, fmt.Errorf("%s:%d: unmatched closing brace", file, t.line)
		}
		if isSpecifier(t.text) {
			if _, err := p.parsePrim(""); err != nil {
				return nil, fmt.Errorf("%s:%d: %w", file, t.line, err)
			}
			continue
		}
		p.skipStatement()
	}
	if p.layer.ID == "" {
		p.layer.ID = layerID
	}
	if p.layer.Strength == 0 {
		return nil, fmt.Errorf("%s: missing positive layerStrength in root metadata", file)
	}
	return p.layer, nil
}

func LoadLayerFiles(files []string) ([]*Layer, error) {
	layers := make([]*Layer, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		id := strings.TrimSuffix(path.Base(file), path.Ext(file))
		layer, err := ParseUSDA(id, filepathSlash(file), data)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

func filepathSlash(name string) string {
	return strings.ReplaceAll(name, string(os.PathSeparator), "/")
}

func (p *usdaParser) eof() bool {
	return p.pos >= len(p.toks)
}

func (p *usdaParser) peek() token {
	if p.eof() {
		return token{kind: tokNewline}
	}
	return p.toks[p.pos]
}

func (p *usdaParser) take() token {
	t := p.peek()
	if !p.eof() {
		p.pos++
	}
	return t
}

func (p *usdaParser) at(text string) bool {
	t := p.peek()
	return t.kind == tokPunct && t.text == text
}

func (p *usdaParser) skipSeparators() {
	for !p.eof() {
		t := p.peek()
		if t.kind == tokNewline || (t.kind == tokPunct && t.text == ";") {
			p.pos++
			continue
		}
		break
	}
}

func (p *usdaParser) skipStatement() {
	depth := 0
	for !p.eof() {
		t := p.take()
		if t.kind == tokPunct {
			switch t.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				if depth == 0 {
					p.pos--
					return
				}
				depth--
			}
		}
		if depth == 0 && t.kind == tokNewline {
			return
		}
	}
}

func (p *usdaParser) parseLayerMetadata() error {
	open := p.take()
	depth := 1
	var body []token
	for depth > 0 {
		if p.eof() {
			return fmt.Errorf("line %d: unterminated layer metadata", open.line)
		}
		t := p.take()
		if t.kind == tokPunct {
			switch t.text {
			case "(":
				depth++
			case ")":
				depth--
				if depth == 0 {
					break
				}
			}
		}
		if depth > 0 {
			body = append(body, t)
		}
	}
	for i := 0; i+2 < len(body); i++ {
		if body[i].kind != tokWord || body[i+1].text != "=" {
			continue
		}
		switch body[i].text {
		case "layerId":
			if body[i+2].kind == tokString || body[i+2].kind == tokWord {
				p.layer.ID = body[i+2].text
			}
		case "layerStrength":
			n, err := strconv.Atoi(body[i+2].text)
			if err != nil {
				return fmt.Errorf("invalid layerStrength %q", body[i+2].text)
			}
			p.layer.Strength = n
		}
	}
	return nil
}

func isSpecifier(s string) bool {
	return s == string(SpecDef) || s == string(SpecOver) || s == string(SpecClass)
}

func joinPrimPath(parent, name string) string {
	if strings.HasPrefix(name, "/") {
		return path.Clean(name)
	}
	if parent == "" {
		return path.Clean("/" + name)
	}
	return path.Clean(parent + "/" + name)
}

func (p *usdaParser) parsePrim(parentPath string) (*PrimSpec, error) {
	head := p.take()
	spec := &PrimSpec{
		Specifier: Specifier(head.text),
		Layer:     p.layer,
		File:      p.layer.File,
		Line:      head.line,
	}
	p.skipSeparators()
	if p.eof() {
		return nil, fmt.Errorf("incomplete %s statement", head.text)
	}
	if p.peek().kind == tokWord && p.peekAt(1).kind == tokString {
		spec.TypeName = p.take().text
		p.skipSeparators()
	}
	nameTok := p.take()
	if nameTok.kind != tokString {
		return nil, fmt.Errorf("%s requires a quoted prim name", head.text)
	}
	spec.Name = nameTok.text
	spec.Path = joinPrimPath(parentPath, spec.Name)
	p.skipSeparators()
	if p.at("(") {
		p.skipBalanced("(", ")")
	}
	p.skipSeparators()
	if !p.at("{") {
		return nil, fmt.Errorf("%s %q has no body", head.text, spec.Name)
	}
	p.take()
	p.layer.Specs = append(p.layer.Specs, spec)

	for {
		p.skipSeparators()
		if p.eof() {
			return nil, fmt.Errorf("%s %q: unterminated body", head.text, spec.Name)
		}
		if p.at("}") {
			p.take()
			return spec, nil
		}
		t := p.peek()
		if isSpecifier(t.text) {
			if _, err := p.parsePrim(spec.Path); err != nil {
				return nil, err
			}
			continue
		}
		if t.kind == tokPunct && t.text == "(" {
			p.skipBalanced("(", ")")
			continue
		}
		if err := p.parseAttribute(spec); err != nil {
			return nil, err
		}
	}
}

func (p *usdaParser) skipBalanced(open, close string) {
	depth := 0
	for !p.eof() {
		t := p.take()
		if t.kind != tokPunct {
			continue
		}
		if t.text == open {
			depth++
		}
		if t.text == close {
			depth--
			if depth == 0 {
				return
			}
		}
	}
}

func (p *usdaParser) peekAt(offset int) token {
	pos := p.pos
	for i := 0; i < offset; i++ {
		if pos >= len(p.toks) {
			return token{kind: tokNewline}
		}
		pos++
		for pos < len(p.toks) && p.toks[pos].kind == tokNewline {
			pos++
		}
	}
	if pos >= len(p.toks) {
		return token{kind: tokNewline}
	}
	return p.toks[pos]
}

func (p *usdaParser) parseAttribute(spec *PrimSpec) error {
	if p.peek().text == "custom" {
		p.take()
	}
	typeTok := p.take()
	if typeTok.kind != tokWord {
		return fmt.Errorf("expected attribute type, got %q", typeTok.text)
	}
	nameTok := p.take()
	if nameTok.kind != tokWord {
		return fmt.Errorf("expected attribute name after %s", typeTok.text)
	}
	p.skipSeparators()
	if !p.at("=") {
		p.skipStatement()
		return nil
	}
	p.take()
	var value []token
	depth := 0
	for !p.eof() {
		t := p.peek()
		if depth == 0 && (t.kind == tokNewline || (t.kind == tokPunct && t.text == "}")) {
			break
		}
		if t.kind == tokPunct {
			switch t.text {
			case "(", "[":
				depth++
			case ")", "]":
				if depth > 0 {
					depth--
				}
			}
		}
		value = append(value, p.take())
	}
	if len(value) == 0 {
		return fmt.Errorf("attribute %s has no value", nameTok.text)
	}
	canonical, color, err := parseAttributeValue(value)
	if err != nil {
		return fmt.Errorf("line %d: attribute %s: %w", nameTok.line, nameTok.text, err)
	}
	spec.Attributes = append(spec.Attributes, &AttributeSpec{
		Name:     nameTok.text,
		TypeName: typeTok.text,
		Value:    canonical,
		Color:    color,
		Line:     nameTok.line,
		Spec:     spec,
		Layer:    p.layer,
		File:     p.layer.File,
	})
	return nil
}

func parseAttributeValue(toks []token) (string, Color3, error) {
	numbers := make([]float64, 0, 3)
	for _, t := range toks {
		if t.kind == tokWord {
			v, err := strconv.ParseFloat(strings.TrimSuffix(t.text, "f"), 64)
			if err != nil {
				return joinTokens(toks), Color3{}, nil
			}
			numbers = append(numbers, v)
		}
	}
	if len(numbers) != 3 {
		return joinTokens(toks), Color3{}, nil
	}
	c := Color3{R: numbers[0], G: numbers[1], B: numbers[2]}
	return fmt.Sprintf("(%s, %s, %s)", formatFloat(c.R), formatFloat(c.G), formatFloat(c.B)), c, nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func joinTokens(toks []token) string {
	var b strings.Builder
	for i, t := range toks {
		if i > 0 && t.kind == tokWord {
			b.WriteByte(' ')
		}
		if t.kind == tokString {
			b.WriteString(strconv.Quote(t.text))
		} else {
			b.WriteString(t.text)
		}
	}
	return b.String()
}
