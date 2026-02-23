package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	errLexerMultilineExpression   = errors.New("lexer error: multiline expressions are not allowed")
	errLexerMissingClosingBrace   = errors.New("lexer error: missing closing '}' for placeholder")
	errLexerPlaceholderPathEmpty  = errors.New("lexer error: placeholder path is empty")
	errLexerEmptyBracketSegment   = errors.New("lexer error: empty bracket segment in path")
	errLexerInvalidIndex          = errors.New("lexer error: invalid index in path")
	errLexerEmptyPathSegment      = errors.New("lexer error: empty path segment")
	errLexerMissingClosingBracket = errors.New("lexer error: missing closing ']' in path")
)

type Lexer struct {
	position      int
	expressionLen int
	expression    string
}

func NewLexer(expression string) *Lexer {
	return &Lexer{
		position:      0,
		expressionLen: len(expression),
		expression:    expression,
	}
}

func (l *Lexer) Lex() ([]token, error) {
	l.position = 0

	tokens := []token{}

	for {
		t, err := l.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return tokens, nil
			}

			return nil, err
		}

		tokens = append(tokens, t)
	}
}

func (l *Lexer) Next() (token, error) {
	if l.position >= l.expressionLen {
		return token{}, io.EOF
	}

	if l.expression[l.position] == '\n' || l.expression[l.position] == '\r' {
		return token{}, errLexerMultilineExpression
	}

	if l.checkPlaceholderStart() {
		start := l.position
		end := strings.IndexByte(l.expression[start:], '}')

		if end < 0 {
			return token{}, errLexerMissingClosingBrace
		}
		end += start

		inner := l.expression[start+2 : end]
		if strings.ContainsAny(inner, "\n\r") {
			return token{}, errLexerMultilineExpression
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
	for l.position < l.expressionLen {
		if l.expression[l.position] == '\n' || l.expression[l.position] == '\r' {
			return token{}, errLexerMultilineExpression
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

func (l *Lexer) checkPlaceholderStart() bool {
	return l.position+1 < l.expressionLen && l.expression[l.position] == '$' && l.expression[l.position+1] == '{'
}

func parsePathSegments(path string) ([]pathSegment, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errLexerPlaceholderPathEmpty
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
				return nil, fmt.Errorf("%w %q", errLexerEmptyBracketSegment, path)
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
					return nil, fmt.Errorf("%w %q in %q", errLexerInvalidIndex, content, path)
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
			return nil, fmt.Errorf("%w in %q", errLexerEmptyPathSegment, path)
		}
		segments = append(segments, pathSegment{
			key:     part,
			index:   0,
			isIndex: false,
		})
	}

	if len(segments) == 0 {
		return nil, errLexerPlaceholderPathEmpty
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

	return -1, fmt.Errorf("%w %q", errLexerMissingClosingBracket, path)
}
