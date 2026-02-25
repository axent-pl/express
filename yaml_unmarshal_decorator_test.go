package express

import (
	"os"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	type ConfigFlat struct {
		Value string `yaml:"value" express:"true"`
	}
	type ConfigNested struct {
		Nested struct {
			Value string `yaml:"value" express:"true"`
		} `yaml:"nested"`
	}
	envs := map[string]string{
		"AXENT_EXPRESS_ENV_TEST": "from-env",
	}
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Setenv() error = %v", err)
		}
	}

	tests := []struct {
		name    string // description of this test case
		input   []byte
		target  any
		options []UnmarshalOption
		wantErr bool
		checker func(any) bool
	}{
		{
			name:    "flat",
			target:  &ConfigFlat{},
			input:   []byte("value: ${AXENT_EXPRESS_ENV_TEST}\n"),
			wantErr: false,
			options: []UnmarshalOption{WithEnv()},
			checker: func(t any) bool {
				c, ok := t.(*ConfigFlat)
				if !ok {
					return false
				}
				return c.Value == "from-env"
			},
		},
		{
			name:    "nested",
			target:  &ConfigNested{},
			input:   []byte("nested: \n  value: ${AXENT_EXPRESS_ENV_TEST}\n"),
			wantErr: false,
			options: []UnmarshalOption{WithEnv()},
			checker: func(t any) bool {
				c, ok := t.(*ConfigNested)
				if !ok {
					return false
				}
				return c.Nested.Value == "from-env"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := Unmarshal(tt.input, tt.target, tt.options...)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Unmarshal() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Unmarshal() succeeded unexpectedly")
			}
			if !tt.checker(tt.target) {
				t.Errorf("Unmarshal() check failed")
			}
		})
	}
}
