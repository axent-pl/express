package express

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type UnmarshalOption func(*unmarshalOptions)

type unmarshalOptions struct {
	withEnv bool
}

func WithEnv() UnmarshalOption {
	return func(opts *unmarshalOptions) {
		opts.withEnv = true
	}
}

func Unmarshal(input []byte, out any, options ...UnmarshalOption) error {
	if err := yaml.Unmarshal(input, out); err != nil {
		return err
	}

	opts := &unmarshalOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	value := reflect.ValueOf(out)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("out must be a non-nil pointer")
	}

	if err := applyExpressions(value, opts); err != nil {
		return err
	}

	return nil
}

func applyExpressions(root reflect.Value, opts *unmarshalOptions) error {
	return walkAndEvaluate(root.Elem(), root, opts)
}

func walkAndEvaluate(current reflect.Value, root reflect.Value, opts *unmarshalOptions) error {
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
				if err := evaluateField(fieldValue, root, opts); err != nil {
					return fmt.Errorf("field %s: %w", fieldType.Name, err)
				}
			}

			if err := walkAndEvaluate(fieldValue, root, opts); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range current.Len() {
			if err := walkAndEvaluate(current.Index(i), root, opts); err != nil {
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

			if err := walkAndEvaluate(mutable, root, opts); err != nil {
				return err
			}

			current.SetMapIndex(key, mutable)
		}
	}

	return nil
}

func evaluateField(field reflect.Value, root reflect.Value, opts *unmarshalOptions) error {
	if !field.CanSet() {
		return nil
	}

	for field.Kind() == reflect.Ptr {
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

	data, err := buildDataMap(root.Elem(), opts)
	if err != nil {
		return err
	}

	result, err := expr.Execute(data)
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

func buildDataMap(root reflect.Value, opts *unmarshalOptions) (map[string]any, error) {
	data, err := toStringMap(root)
	if err != nil {
		return nil, err
	}

	if opts != nil && opts.withEnv {
		for _, pair := range os.Environ() {
			key, value, found := strings.Cut(pair, "=")
			if !found {
				continue
			}

			data[key] = value
		}
	}

	return data, nil
}

func toStringMap(value reflect.Value) (map[string]any, error) {
	if !value.IsValid() {
		return map[string]any{}, nil
	}

	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return map[string]any{}, nil
		}
		value = value.Elem()
	}

	resultAny, err := exportValue(value)
	if err != nil {
		return nil, err
	}

	resultMap, ok := resultAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("root value must resolve to map[string]any")
	}

	return resultMap, nil
}

func exportValue(value reflect.Value) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}

	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		out := make(map[string]any)
		valueType := value.Type()

		for i := range value.NumField() {
			fieldType := valueType.Field(i)
			if fieldType.PkgPath != "" {
				continue
			}

			name, include := yamlFieldName(fieldType)
			if !include {
				continue
			}

			fieldAny, err := exportValue(value.Field(i))
			if err != nil {
				return nil, err
			}

			out[name] = fieldAny
		}

		return out, nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map keys must be strings")
		}

		out := make(map[string]any)
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			elemAny, err := exportValue(iter.Value())
			if err != nil {
				return nil, err
			}

			out[key] = elemAny
		}

		return out, nil
	case reflect.Slice, reflect.Array:
		length := value.Len()
		out := make([]any, length)
		for i := range length {
			elemAny, err := exportValue(value.Index(i))
			if err != nil {
				return nil, err
			}

			out[i] = elemAny
		}

		return out, nil
	default:
		return value.Interface(), nil
	}
}

func yamlFieldName(field reflect.StructField) (name string, include bool) {
	tag := field.Tag.Get("yaml")
	if tag == "-" {
		return "", false
	}

	if tag == "" {
		return field.Name, true
	}

	base, _, _ := strings.Cut(tag, ",")
	if base == "" {
		return field.Name, true
	}

	return base, true
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
