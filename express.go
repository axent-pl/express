package main

import (
	"fmt"
	"reflect"
	"strings"
)

type Expression struct {
	expression string
	exec       func(data map[string]any) (any, error)
}

func Compile(expression string) (*Expression, error) {
	l := NewLexer(expression)
	tokens, err := l.Lex()
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
