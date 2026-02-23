package main

import (
	"strings"
	"testing"
)

func TestLexerLexSuccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []token
	}{
		{
			name:  "literal only",
			input: "hello",
			want: []token{{
				kind:    tokenLiteral,
				literal: "hello",
				raw:     "hello",
			}},
		},
		{
			name:  "single placeholder",
			input: "${user.name}",
			want: []token{{
				kind: tokenPlaceholder,
				segments: []pathSegment{
					{key: "user"},
					{key: "name"},
				},
				raw: "${user.name}",
			}},
		},
		{
			name:  "placeholder with index and quoted key and default",
			input: `${users[1]["display-name"]|anon}`,
			want: []token{{
				kind: tokenPlaceholder,
				segments: []pathSegment{
					{key: "users"},
					{index: 1, isIndex: true},
					{key: "display-name"},
				},
				hasDefault: true,
				def:        "anon",
				raw:        `${users[1]["display-name"]|anon}`,
			}},
		},
		{
			name:  "mixed literal and placeholders",
			input: "x=${a} y=${b|2}",
			want: []token{
				{kind: tokenLiteral, literal: "x=", raw: "x="},
				{kind: tokenPlaceholder, segments: []pathSegment{{key: "a"}}, raw: "${a}"},
				{kind: tokenLiteral, literal: " y=", raw: " y="},
				{kind: tokenPlaceholder, segments: []pathSegment{{key: "b"}}, hasDefault: true, def: "2", raw: "${b|2}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			got, err := l.Lex()
			if err != nil {
				t.Fatalf("Lex() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Lex() token count = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i].kind != tt.want[i].kind {
					t.Fatalf("token[%d].kind = %v, want %v", i, got[i].kind, tt.want[i].kind)
				}
				if got[i].literal != tt.want[i].literal {
					t.Fatalf("token[%d].literal = %q, want %q", i, got[i].literal, tt.want[i].literal)
				}
				if got[i].raw != tt.want[i].raw {
					t.Fatalf("token[%d].raw = %q, want %q", i, got[i].raw, tt.want[i].raw)
				}
				if got[i].hasDefault != tt.want[i].hasDefault {
					t.Fatalf("token[%d].hasDefault = %v, want %v", i, got[i].hasDefault, tt.want[i].hasDefault)
				}
				if got[i].def != tt.want[i].def {
					t.Fatalf("token[%d].def = %q, want %q", i, got[i].def, tt.want[i].def)
				}
				if len(got[i].segments) != len(tt.want[i].segments) {
					t.Fatalf("token[%d].segments len = %d, want %d", i, len(got[i].segments), len(tt.want[i].segments))
				}
				for j := range tt.want[i].segments {
					gs, ws := got[i].segments[j], tt.want[i].segments[j]
					if gs.key != ws.key || gs.index != ws.index || gs.isIndex != ws.isIndex {
						t.Fatalf("token[%d].segments[%d] = %#v, want %#v", i, j, gs, ws)
					}
				}
			}
		})
	}
}

func TestLexerLexErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "missing closing brace", input: "${user.name", wantErr: "missing closing '}'"},
		{name: "empty placeholder path", input: "${   }", wantErr: "placeholder path is empty"},
		{name: "invalid index", input: "${users[abc]}", wantErr: "invalid index"},
		{name: "missing closing bracket", input: "${users[0}", wantErr: "missing closing ']'"},
		{name: "multiline expression", input: "a\nb", wantErr: "multiline expressions are not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLexer(tt.input).Lex()
			if err == nil {
				t.Fatalf("Lex() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Lex() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
