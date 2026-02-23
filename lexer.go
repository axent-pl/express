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

	if strings.ContainsAny(l.expression, "\n\r") {
		return tokens, errLexerMultilineExpression
	}

	for {
		t, err := l.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return tokens, nil
			}

			return nil, err
		}

		tokens = append(tokens, t)
	}
}

func (l *Lexer) next() (token, error) {
	if l.position >= l.expressionLen {
		return token{}, io.EOF
	}

	if l.checkPlaceholderStart() {
		start := l.position
		end := strings.IndexByte(l.expression[start:], '}')

		if end < 0 {
			return token{}, errLexerMissingClosingBrace
		}

		end += start

		inner := l.expression[start+2 : end]

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
		if l.checkPlaceholderStart() {
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
			end, isQuoted, err := parseBracketIndex(path, i)
			if err != nil {
				return nil, err
			}

			content := strings.TrimSpace(path[i+1 : end])
			if content == "" {
				return nil, fmt.Errorf("%w %q", errLexerEmptyBracketSegment, path)
			}

			if isQuoted {
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

func parseBracketIndex(path string, start int) (endIndex int, isQuoted bool, err error) {
	var openingQuote byte

	if start+1 < len(path) {
		if q := path[start+1]; q == '"' || q == '\'' {
			openingQuote = q
		}
	}

	for i := start + 1; i < len(path); i++ {
		ch := path[i]

		if ch == ']' {
			if openingQuote != 0 && i-1 > start && path[i-1] == openingQuote {
				return i, true, nil
			}
			return i, false, nil
		}
	}

	return -1, false, fmt.Errorf("%w %q", errLexerMissingClosingBracket, path)
}
