package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrParse = errors.New("expression parser error")

func ParseTemplate(input string) ([]token, error) {
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
				return nil, fmt.Errorf("%w: unterminated placeholder at position %d", ErrParse, i)
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
		return token{}, fmt.Errorf("%w: empty placeholder", ErrParse)
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
		return nil, fmt.Errorf("%w: placeholder path is empty", ErrParse)
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
				return nil, fmt.Errorf("%w: missing closing ']' in path %q", ErrParse, path)
			}
			end += i
			content := strings.TrimSpace(path[i+1 : end])
			if content == "" {
				return nil, fmt.Errorf("%w: empty bracket segment in path %q", ErrParse, path)
			}

			if len(content) >= 2 && ((content[0] == '\'' && content[len(content)-1] == '\'') || (content[0] == '"' && content[len(content)-1] == '"')) {
				segments = append(segments, pathSegment{key: content[1 : len(content)-1]})
			} else {
				idx, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("%w: invalid index %q in path %q", ErrParse, content, path)
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
			return nil, fmt.Errorf("%w: empty path segment in %q", ErrParse, path)
		}
		segments = append(segments, pathSegment{key: part})
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("%w, path %q has no segments", ErrParse, path)
	}

	return segments, nil
}
