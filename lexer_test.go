package express

import (
	"strings"
	"testing"
)

func TestLexer_Lex(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		want           []token
		wantErr        bool
		wantErrMessage string
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
		{
			name:           "missing closing brace",
			input:          "${user.name",
			wantErrMessage: "missing closing '}'",
			wantErr:        true,
		},
		{
			name:           "empty placeholder path",
			input:          "${   }",
			wantErrMessage: "placeholder path is empty",
			wantErr:        true,
		},
		{
			name:           "invalid index",
			input:          "${users[abc]}",
			wantErrMessage: "invalid index",
			wantErr:        true,
		},
		{
			name:           "missing closing bracket",
			input:          "${users[0}",
			wantErrMessage: "missing closing ']'",
			wantErr:        true,
		},
		{
			name:           "multiline expression",
			input:          "a\nb",
			wantErrMessage: "multiline expressions are not allowed",
			wantErr:        true,
		},
		{
			name:           "empty bracket segment",
			input:          "${users[]}",
			wantErrMessage: "empty bracket",
			wantErr:        true,
		},
		{
			name:           "empty bracket segment (space)",
			input:          "${users[ ]}",
			wantErrMessage: "empty bracket",
			wantErr:        true,
		},
		{
			name:           "empty path segment",
			input:          "${}",
			wantErrMessage: "placeholder path is empty",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			got, gotErr := l.Lex()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Lex() failed: %v", gotErr)
				}
				if !strings.Contains(gotErr.Error(), tt.wantErrMessage) {
					t.Fatalf("Lex() error = %q, want substring %q", gotErr.Error(), tt.wantErrMessage)
				}
			} else if tt.wantErr {
				t.Fatalf("Lex() error = nil, want %s", tt.wantErrMessage)
			} else {
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
			}
		})
	}
}
