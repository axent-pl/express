package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ErrParse = errors.New("expression parser error")

var tokenizerRE = regexp.MustCompile(`\$\{([^$}|]+)(?:\|([^}]+))?\}`)

func ParseTemplate(input string) ([]token, error) {
	var out []token

	matches := tokenizerRE.FindAllStringSubmatchIndex(input, -1)
	prev := 0

	for _, m := range matches {
		start, end := m[0], m[1]

		// literal before placeholder
		if start > prev {
			out = append(out, token{
				kind:    tokenLiteral,
				literal: input[prev:start],
			})
		}

		// placeholder
		placeholder, parseErr := parsePlaceholder(input, m)
		if parseErr != nil {
			return nil, parseErr
		}
		out = append(out, placeholder)

		prev = end
	}

	// trailing literal
	if prev < len(input) {
		out = append(out, token{
			literal: input[prev:],
			kind:    tokenLiteral,
		})
	}

	return out, nil
}

func parsePlaceholder(input string, m []int) (token, error) {
	pathStart, pathEnd := m[2], m[3]
	defStart, defEnd := m[4], m[5]
	pathExpr := input[pathStart:pathEnd]

	def := ""
	hasDefault := defStart > -1 && defEnd > -1
	if hasDefault {
		def = input[defStart:defEnd]
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
