package express

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type UnmarshalOption func(*unmarshalContext)

type unmarshalContext struct {
	data map[string]any
}

func WithEnv() UnmarshalOption {
	return func(opts *unmarshalContext) {
		for _, pair := range os.Environ() {
			key, value, found := strings.Cut(pair, "=")
			if !found {
				continue
			}
			opts.data[key] = value
		}
	}
}

func Unmarshal(input []byte, out any, options ...UnmarshalOption) error {
	if err := yaml.Unmarshal(input, out); err != nil {
		return err
	}

	opts := &unmarshalContext{
		data: map[string]any{},
	}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	value := reflect.ValueOf(out)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("out must be a non-nil pointer")
	}

	if err := applyExpressions(value, opts); err != nil {
		return err
	}

	return nil
}

func applyExpressions(root reflect.Value, opts *unmarshalContext) error {
	return walkAndEvaluate(root.Elem(), opts)
}

func walkAndEvaluate(current reflect.Value, opts *unmarshalContext) error {
	if !current.IsValid() {
		return nil
	}

	for current.Kind() == reflect.Ptr {
		if current.IsNil() {
			return nil
		}
		current = current.Elem()
	}

	switch current.Kind() {
	case reflect.Struct:
		currentType := current.Type()
		for i := range current.NumField() {
			fieldType := currentType.Field(i)
			fieldValue := current.Field(i)

			if fieldType.PkgPath != "" {
				continue
			}

			if _, ok := fieldType.Tag.Lookup("express"); ok {
				if err := evaluateField(fieldValue, opts); err != nil {
					return fmt.Errorf("field %s: %w", fieldType.Name, err)
				}
			}

			if err := walkAndEvaluate(fieldValue, opts); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range current.Len() {
			if err := walkAndEvaluate(current.Index(i), opts); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := current.MapRange()
		for iter.Next() {
			key := iter.Key()
			elem := iter.Value()

			mutable := reflect.New(elem.Type()).Elem()
			mutable.Set(elem)

			if err := walkAndEvaluate(mutable, opts); err != nil {
				return err
			}

			current.SetMapIndex(key, mutable)
		}
	}

	return nil
}

func evaluateField(field reflect.Value, opts *unmarshalContext) error {
	if !field.CanSet() {
		return nil
	}

	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil
		}
		field = field.Elem()
	}

	if field.Kind() != reflect.String {
		return fmt.Errorf("express tag requires string field")
	}

	expr, err := Compile(field.String())
	if err != nil {
		return err
	}

	result, err := expr.Execute(opts.data)
	if err != nil {
		return err
	}

	converted, err := convertValue(result, field.Type())
	if err != nil {
		return err
	}

	field.Set(converted)

	return nil
}

func convertValue(input any, target reflect.Type) (reflect.Value, error) {
	if input == nil {
		return reflect.Zero(target), nil
	}

	value := reflect.ValueOf(input)
	if value.Type().AssignableTo(target) {
		return value, nil
	}

	if value.Type().ConvertibleTo(target) {
		return value.Convert(target), nil
	}

	if target.Kind() == reflect.String {
		return reflect.ValueOf(fmt.Sprint(input)), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot assign %T to %s", input, target.String())
}
