package express

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	errExpressionNotCompiled = errors.New("expression is not compiled")
	errValueNil              = errors.New("value is nil")
	errUnknownTokenKind      = errors.New("unknown token kind")
	errValueNotArray         = errors.New("value is not an array")
	errArrayIndexOutOfRange  = errors.New("array index out of range")
	errValueNotObject        = errors.New("value is not an object")
	errMissingKey            = errors.New("missing key")
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
		return nil, errExpressionNotCompiled
	}
	return e.exec(data)
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
				return nil, fmt.Errorf("%w for %q", errValueNil, placeholder.raw)
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
					return nil, fmt.Errorf("%w for %q", errValueNil, t.raw)
				}
				fmt.Fprint(&b, value)
			default:
				return nil, errUnknownTokenKind
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
				return nil, errValueNotArray
			}
			if segment.index < 0 || segment.index >= v.Len() {
				return nil, fmt.Errorf("%w: %d", errArrayIndexOutOfRange, segment.index)
			}
			current = v.Index(segment.index).Interface()
			continue
		}

		v := reflect.ValueOf(current)
		if !v.IsValid() || v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("%w for key %q", errValueNotObject, segment.key)
		}

		next := v.MapIndex(reflect.ValueOf(segment.key))
		if !next.IsValid() {
			return nil, fmt.Errorf("%w %q", errMissingKey, segment.key)
		}
		current = next.Interface()
	}

	return current, nil
}
