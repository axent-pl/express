package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Lexer struct {
	position   int
	expression string
}

func NewLexer(expression string) *Lexer {
	return &Lexer{
		position:   0,
		expression: expression,
	}
}

func (l *Lexer) Lex() ([]token, error) {
	l.position = 0

	tokens := []token{}
	for {
		t, err := l.Next()
		switch err {
		case nil:
			tokens = append(tokens, t)
		case io.EOF:
			return tokens, nil
		default:
			return nil, err
		}
	}
}

func (l *Lexer) Next() (token, error) {
	if l.position >= len(l.expression) {
		return token{}, io.EOF
	}

	if l.expression[l.position] == '\n' || l.expression[l.position] == '\r' {
		return token{}, fmt.Errorf("lexer error: multiline expressions are not allowed")
	}

	if strings.HasPrefix(l.expression[l.position:], "${") {
		start := l.position
		end := strings.IndexByte(l.expression[start:], '}')
		if end < 0 {
			return token{}, fmt.Errorf("lexer error: missing closing '}' for placeholder")
		}
		end += start

		inner := l.expression[start+2 : end]
		if strings.ContainsAny(inner, "\n\r") {
			return token{}, fmt.Errorf("lexer error: multiline expressions are not allowed")
		}

		pathExpr := inner
		def := ""
		hasDefault := false
		if before, after, ok := strings.Cut(inner, "|"); ok {
			pathExpr = before
			def = after
			hasDefault = true
		}

		segments, err := parsePathSegments(pathExpr)
		if err != nil {
			return token{}, err
		}

		l.position = end + 1

		raw := l.expression[start : end+1]
		return token{
			kind:       tokenPlaceholder,
			segments:   segments,
			hasDefault: hasDefault,
			def:        def,
			raw:        raw,
		}, nil
	}

	start := l.position
	for l.position < len(l.expression) {
		if l.expression[l.position] == '\n' || l.expression[l.position] == '\r' {
			return token{}, fmt.Errorf("lexer error: multiline expressions are not allowed")
		}
		if strings.HasPrefix(l.expression[l.position:], "${") {
			break
		}
		l.position++
	}

	return token{
		kind:    tokenLiteral,
		literal: l.expression[start:l.position],
		raw:     l.expression[start:l.position],
	}, nil
}

func parsePathSegments(path string) ([]pathSegment, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("lexer error: placeholder path is empty")
	}

	segments := make([]pathSegment, 0)
	i := 0
	for i < len(path) {
		if path[i] == '.' {
			i++
			continue
		}

		if path[i] == '[' {
			end, err := findClosingBracket(path, i)
			if err != nil {
				return nil, err
			}

			content := strings.TrimSpace(path[i+1 : end])
			if content == "" {
				return nil, fmt.Errorf("lexer error: empty bracket segment in path %q", path)
			}

			if len(content) >= 2 && ((content[0] == '"' && content[len(content)-1] == '"') || (content[0] == '\'' && content[len(content)-1] == '\'')) {
				segments = append(segments, pathSegment{
					key:     content[1 : len(content)-1],
					index:   0,
					isIndex: false,
				})
			} else {
				idx, parseErr := strconv.Atoi(content)
				if parseErr != nil {
					return nil, fmt.Errorf("lexer error: invalid index %q in path %q", content, path)
				}
				segments = append(segments, pathSegment{
					key:     "",
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
			return nil, fmt.Errorf("lexer error: empty path segment in %q", path)
		}
		segments = append(segments, pathSegment{
			key:     part,
			index:   0,
			isIndex: false,
		})
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("lexer error: placeholder path is empty")
	}

	return segments, nil
}

func findClosingBracket(path string, start int) (int, error) {
	quote := byte(0)

	for i := start + 1; i < len(path); i++ {
		ch := path[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}

		if ch == ']' {
			return i, nil
		}
	}

	return -1, fmt.Errorf("lexer error: missing closing ']' in path %q", path)
}
