package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Expression struct {
	expression string
	exec       func(data map[string]any) (any, error)
}

func Compile(expression string) (*Expression, error) {
	tokens, err := parseTemplate(expression)
	if err != nil {
		return nil, err
	}

	exec, err := buildExecutor(tokens)
	if err != nil {
		return nil, err
	}

	e := &Expression{
		expression: expression,
		exec:       exec,
	}
	return e, nil
}

// extracts value from data based on the expression
// Supports access by key, defaults, arrays, for string values even multiple placeholders....
// ${someKey.innerKey}
// ${someKey.innerKey|default}
// ${someKey.innerKey}/${someKey.otherKey|otherdefault}
func (e *Expression) Execute(data map[string]any) (any, error) {
	if e == nil || e.exec == nil {
		return nil, fmt.Errorf("expression is not compiled")
	}
	return e.exec(data)
}

type tokenKind int

const (
	tokenLiteral tokenKind = iota
	tokenPlaceholder
)

type pathSegment struct {
	key     string
	index   int
	isIndex bool
}

type token struct {
	kind       tokenKind
	literal    string
	segments   []pathSegment
	hasDefault bool
	def        string
	raw        string
}

func parseTemplate(input string) ([]token, error) {
	var out []token
	var literal strings.Builder

	for i := 0; i < len(input); i++ {
		if i+1 < len(input) && input[i] == '$' && input[i+1] == '{' {
			if literal.Len() > 0 {
				out = append(out, token{
					kind:    tokenLiteral,
					literal: literal.String(),
				})
				literal.Reset()
			}

			end := strings.IndexByte(input[i+2:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unterminated placeholder at position %d", i)
			}
			end += i + 2

			raw := input[i+2 : end]
			placeholder, err := parsePlaceholder(raw)
			if err != nil {
				return nil, err
			}
			out = append(out, placeholder)
			i = end
			continue
		}

		literal.WriteByte(input[i])
	}

	if literal.Len() > 0 {
		out = append(out, token{
			kind:    tokenLiteral,
			literal: literal.String(),
		})
	}

	if len(out) == 0 {
		out = append(out, token{
			kind:    tokenLiteral,
			literal: "",
		})
	}

	return out, nil
}

func parsePlaceholder(raw string) (token, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return token{}, fmt.Errorf("empty placeholder")
	}

	pathExpr := content
	def := ""
	hasDefault := false

	if idx := strings.IndexByte(content, '|'); idx >= 0 {
		pathExpr = strings.TrimSpace(content[:idx])
		def = content[idx+1:]
		hasDefault = true
	}

	segments, err := parsePath(pathExpr)
	if err != nil {
		return token{}, err
	}

	return token{
		kind:       tokenPlaceholder,
		segments:   segments,
		hasDefault: hasDefault,
		def:        def,
		raw:        content,
	}, nil
}

func parsePath(path string) ([]pathSegment, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("placeholder path is empty")
	}

	var segments []pathSegment
	i := 0

	for i < len(path) {
		if path[i] == '.' {
			i++
			continue
		}

		if path[i] == '[' {
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("missing closing ']' in path %q", path)
			}
			end += i
			content := strings.TrimSpace(path[i+1 : end])
			if content == "" {
				return nil, fmt.Errorf("empty bracket segment in path %q", path)
			}

			if len(content) >= 2 && ((content[0] == '\'' && content[len(content)-1] == '\'') || (content[0] == '"' && content[len(content)-1] == '"')) {
				segments = append(segments, pathSegment{key: content[1 : len(content)-1]})
			} else {
				idx, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("invalid index %q in path %q", content, path)
				}
				segments = append(segments, pathSegment{
					index:   idx,
					isIndex: true,
				})
			}

			i = end + 1
			continue
		}

		start := i
		for i < len(path) && path[i] != '.' && path[i] != '[' {
			i++
		}
		part := strings.TrimSpace(path[start:i])
		if part == "" {
			return nil, fmt.Errorf("empty path segment in %q", path)
		}
		segments = append(segments, pathSegment{key: part})
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("path %q has no segments", path)
	}

	return segments, nil
}

func buildExecutor(tokens []token) (func(data map[string]any) (any, error), error) {
	if len(tokens) == 1 && tokens[0].kind == tokenLiteral {
		lit := tokens[0].literal
		return func(_ map[string]any) (any, error) {
			return lit, nil
		}, nil
	}

	if len(tokens) == 1 && tokens[0].kind == tokenPlaceholder {
		placeholder := tokens[0]
		return func(data map[string]any) (any, error) {
			value, err := resolvePath(data, placeholder.segments)
			if err != nil || value == nil {
				if placeholder.hasDefault {
					return placeholder.def, nil
				}
				if err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("value for %q is nil", placeholder.raw)
			}
			return value, nil
		}, nil
	}

	return func(data map[string]any) (any, error) {
		var b strings.Builder

		for _, t := range tokens {
			switch t.kind {
			case tokenLiteral:
				b.WriteString(t.literal)
			case tokenPlaceholder:
				value, err := resolvePath(data, t.segments)
				if err != nil || value == nil {
					if t.hasDefault {
						b.WriteString(t.def)
						continue
					}
					if err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("value for %q is nil", t.raw)
				}
				b.WriteString(fmt.Sprint(value))
			default:
				return nil, fmt.Errorf("unknown token kind")
			}
		}

		return b.String(), nil
	}, nil
}

func resolvePath(data map[string]any, segments []pathSegment) (any, error) {
	var current any = data

	for _, segment := range segments {
		if segment.isIndex {
			v := reflect.ValueOf(current)
			if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
				return nil, fmt.Errorf("value is not an array")
			}
			if segment.index < 0 || segment.index >= v.Len() {
				return nil, fmt.Errorf("array index out of range: %d", segment.index)
			}
			current = v.Index(segment.index).Interface()
			continue
		}

		v := reflect.ValueOf(current)
		if !v.IsValid() || v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("value is not an object for key %q", segment.key)
		}

		next := v.MapIndex(reflect.ValueOf(segment.key))
		if !next.IsValid() {
			return nil, fmt.Errorf("missing key %q", segment.key)
		}
		current = next.Interface()
	}

	return current, nil
}
