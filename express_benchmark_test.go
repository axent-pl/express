package main

import (
	"strings"
	"testing"
)

func BenchmarkParseTemplate(b *testing.B) {
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
				if _, err := ParseTemplate(tc.input); err != nil {
					b.Fatalf("ParseTemplate() error = %v", err)
				}
			}
		})
	}
}
