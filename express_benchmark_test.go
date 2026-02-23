package express

import (
	"strings"
	"testing"
)

var benchmarkExecuteResult any

func BenchmarkLexer_Lex(b *testing.B) {
	b.ReportAllocs()

	benchmarks := []struct {
		name  string
		input string
	}{
		{
			name:  "literal_only",
			input: "just plain text without placeholders",
		},
		{
			name:  "single_placeholder",
			input: "${user.name}",
		},
		{
			name:  "single_placeholder_with_default",
			input: "${user.name|anonymous}",
		},
		{
			name:  "mixed_literals_and_placeholders",
			input: "Hello ${user.name}, role=${user.role|guest}, id=${user.id}",
		},
		{
			name:  "deep_path_with_indexes",
			input: "${org.teams[2].members[10].profile[\"display-name\"]|n/a}",
		},
		{
			name:  "many_placeholders",
			input: strings.Repeat("${a.b[0]|x}-", 20),
		},
	}

	for _, tc := range benchmarks {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l := NewLexer(tc.input)
				if _, err := l.Lex(); err != nil {
					b.Fatalf("Lexer.Lex() error = %v", err)
				}
			}
		})
	}
}

func BenchmarkExecute(b *testing.B) {
	b.ReportAllocs()

	data := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"role": "admin",
			"id":   42,
		},
		"org": map[string]any{
			"teams": []any{
				map[string]any{
					"members": []any{
						map[string]any{
							"profile": map[string]any{
								"display-name": "User0",
							},
						},
					},
				},
				map[string]any{
					"members": []any{
						map[string]any{
							"profile": map[string]any{
								"display-name": "User1",
							},
						},
					},
				},
				map[string]any{
					"members": []any{
						map[string]any{
							"profile": map[string]any{
								"display-name": "Primary User",
							},
						},
					},
				},
			},
		},
	}

	benchmarks := []struct {
		name       string
		expression string
	}{
		{
			name:       "literal_only",
			expression: "just plain text without placeholders",
		},
		{
			name:       "single_placeholder",
			expression: "${user.name}",
		},
		{
			name:       "single_placeholder_with_default",
			expression: "${user.missing|anonymous}",
		},
		{
			name:       "mixed_literals_and_placeholders",
			expression: "Hello ${user.name}, role=${user.role|guest}, id=${user.id}",
		},
		{
			name:       "deep_path_with_indexes",
			expression: "${org.teams[2].members[0].profile[\"display-name\"]|n/a}",
		},
	}

	for _, tc := range benchmarks {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			expr, err := Compile(tc.expression)
			if err != nil {
				b.Fatalf("Compile() error = %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := expr.Execute(data)
				if err != nil {
					b.Fatalf("Execute() error = %v", err)
				}
				benchmarkExecuteResult = result
			}
		})
	}
}
