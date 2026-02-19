package main

import "testing"

func TestExecuteSuccessCases(t *testing.T) {
	data := map[string]any{
		"someKey": map[string]any{
			"innerKey": "value",
			"otherKey": "other",
		},
		"count": 5,
		"users": []any{
			map[string]any{"name": "alice"},
			map[string]any{"name": "bob"},
		},
		"meta": map[string]any{
			"kebab-key": "ok",
		},
		"numbers": []int{10, 20, 30},
		"labels":  map[string]string{"a": "alpha"},
	}

	tests := []struct {
		name       string
		expression string
		want       any
	}{
		{
			name:       "literal only",
			expression: "just text",
			want:       "just text",
		},
		{
			name:       "single placeholder returns raw type",
			expression: "${count}",
			want:       5,
		},
		{
			name:       "nested key",
			expression: "${someKey.innerKey}",
			want:       "value",
		},
		{
			name:       "array index",
			expression: "${users[1].name}",
			want:       "bob",
		},
		{
			name:       "quoted bracket key",
			expression: `${meta["kebab-key"]}`,
			want:       "ok",
		},
		{
			name:       "typed slice index",
			expression: "${numbers[1]}",
			want:       20,
		},
		{
			name:       "typed map key",
			expression: "${labels.a}",
			want:       "alpha",
		},
		{
			name:       "default when missing",
			expression: "${missing.value|fallback}",
			want:       "fallback",
		},
		{
			name:       "mixed placeholders render string",
			expression: "${someKey.innerKey}/${someKey.otherKey|default}",
			want:       "value/other",
		},
		{
			name:       "mixed placeholders with default",
			expression: "prefix-${missing.value|fallback}-suffix",
			want:       "prefix-fallback-suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Compile(tt.expression)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			got, err := expr.Execute(data)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Execute() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExecuteErrors(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		data       map[string]any
	}{
		{
			name:       "missing key without default",
			expression: "${missing.value}",
			data:       map[string]any{},
		},
		{
			name:       "nil value without default",
			expression: "${someKey.innerKey}",
			data: map[string]any{
				"someKey": map[string]any{"innerKey": nil},
			},
		},
		{
			name:       "array index out of range",
			expression: "${users[1]}",
			data: map[string]any{
				"users": []any{"only-one"},
			},
		},
		{
			name:       "index access on non-array",
			expression: "${users[0]}",
			data: map[string]any{
				"users": map[string]any{"0": "not-an-array"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Compile(tt.expression)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			if _, err := expr.Execute(tt.data); err == nil {
				t.Fatal("Execute() error = nil, want non-nil")
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	tests := []struct {
		name       string
		expression string
	}{
		{
			name:       "unterminated placeholder",
			expression: "${someKey.innerKey",
		},
		{
			name:       "empty placeholder",
			expression: "${ }",
		},
		{
			name:       "invalid index",
			expression: "${users[abc]}",
		},
		{
			name:       "missing closing bracket",
			expression: "${users[0}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Compile(tt.expression); err == nil {
				t.Fatal("Compile() error = nil, want non-nil")
			}
		})
	}
}

func TestExecuteNotCompiled(t *testing.T) {
	var expr Expression
	if _, err := expr.Execute(nil); err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
}
