package gonfiguration

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

//nolint:gochecknoglobals
var (
	gonfig     *gonfiguration
	gonfigOnce sync.Once
)

//nolint:gochecknoinits
func init() {
	gonfigOnce.Do(func() {
		gonfig = &gonfiguration{
			defaults: map[string]any{},
			envVars:  map[string]string{},
		}
	})
}

func Parse(dst any) error {
	envVars, err := getEnvVars()
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	gonfig.setEnvVars(envVars)

	dstVal, err := getDstStructValue(dst)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	if err := parseDstFields(dstVal, envVars); err != nil {
		return fmt.Errorf("failed to parse fields: %w", err)
	}

	return nil
}

func MustParse(dst any) {
	if err := Parse(dst); err != nil {
		panic(err)
	}
}

func GetAllValues() map[string]any {
	defaults := gonfig.getDefaults()
	envVars := gonfig.getEnvVars()

	allValues := map[string]any{}

	maps.Copy(allValues, defaults)

	for key, val := range envVars {
		allValues[key] = val
	}

	return allValues
}

func Reset() {
	gonfig.reset()
}

type gonfiguration struct {
	sync.RWMutex
	defaults map[string]any
	envVars  map[string]string
}

func (g *gonfiguration) reset() {
	g.Lock()
	defer g.Unlock()

	gonfig = &gonfiguration{
		defaults: map[string]any{},
		envVars:  map[string]string{},
	}
}

func parseDstFields(
	dstVal reflect.Value,
	envVars map[string]string,
) error {
	for i := range dstVal.NumField() {
		fieldType := dstVal.Type().Field(i)

		tag, ok := fieldType.Tag.Lookup("env")
		if !ok {
			continue
		}

		key, required := parseTag(tag)
		tagDefault := tagDefaultFromField(fieldType)

		fieldValue := dstVal.Field(i)
		if !isSupportedType(fieldValue) {
			return ErrUnsupportedFieldType
		}

		if err := fillFieldValue(fieldValue, key, required, envVars, tagDefault); err != nil {
			return fmt.Errorf("failed to set field value: %w", err)
		}
	}

	return nil
}

func parseTag(tag string) (string, bool) {
	parts := strings.Split(tag, ",")
	key := strings.TrimSpace(parts[0])

	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "required" {
			return key, true
		}
	}

	return key, false
}

func tagDefaultFromField(field reflect.StructField) *string {
	val, ok := field.Tag.Lookup("default")
	if !ok {
		return nil
	}

	return &val
}

func fillFieldValue(
	fieldValue reflect.Value,
	key string,
	required bool,
	envVars map[string]string,
	tagDefault *string,
) error {
	// Tag default has lowest priority
	if tagDefault != nil {
		if err := setEnvVarValue(fieldValue, *tagDefault); err != nil {
			return fmt.Errorf("field %s: invalid default tag value %q: %w", key, *tagDefault, err)
		}
	}

	// Programmatic default overrides tag default
	hasDefault, err := setDefaultValue(fieldValue, key)
	if err != nil {
		return err
	}

	if !hasDefault {
		hasDefault = tagDefault != nil
	}

	// Env var has highest priority
	envVal, hasEnvVar := envVars[key]
	if !hasEnvVar {
		if required && !hasDefault {
			return fmt.Errorf("field %s: %w", key, ErrRequiredFieldNotSet)
		}

		return nil
	}

	return setEnvVarValue(fieldValue, envVal)
}

func setDefaultValue(
	fieldValue reflect.Value,
	key string,
) (bool, error) {
	defaultValue := gonfig.getDefault(key)
	if defaultValue == nil {
		return false, nil
	}

	if reflect.TypeOf(defaultValue) != fieldValue.Type() {
		return false, ErrDefaultTypeMismatch
	}

	fieldValue.Set(reflect.ValueOf(defaultValue))

	return true, nil
}

func setEnvVarValue(
	fieldValue reflect.Value,
	envVal string,
) error {
	// Handle time.Duration specifically since it has underlying type int64
	if fieldValue.Type() == reflect.TypeFor[time.Duration]() {
		return setDuration(fieldValue, envVal)
	}

	// Handle []string specifically
	if fieldValue.Type() == reflect.TypeFor[[]string]() {
		return setStringSlice(fieldValue, envVal)
	}

	switch fieldValue.Kind() { //nolint:exhaustive
	case reflect.String:
		fieldValue.SetString(envVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setInt(fieldValue, envVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUint(fieldValue, envVal)
	case reflect.Float32, reflect.Float64:
		return setFloat(fieldValue, envVal)
	case reflect.Bool:
		return setBool(fieldValue, envVal)
	default:
		return fmt.Errorf("FieldName: %s FieldType %s: %w", fieldValue.Type(), fieldValue.Kind(), ErrUnsupportedFieldType)
	}

	return nil
}

func setInt(
	fieldValue reflect.Value,
	envVal string,
) error {
	num, err := strconv.ParseInt(envVal, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse int: %w", err)
	}

	fieldValue.SetInt(num)

	return nil
}

func setUint(
	fieldValue reflect.Value,
	envVal string,
) error {
	num, err := strconv.ParseUint(envVal, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse uint: %w", err)
	}

	fieldValue.SetUint(num)

	return nil
}

func setFloat(
	fieldValue reflect.Value,
	envVal string,
) error {
	num, err := strconv.ParseFloat(envVal, fieldValue.Type().Bits())
	if err != nil {
		return fmt.Errorf("failed to parse float: %w", err)
	}

	fieldValue.SetFloat(num)

	return nil
}

func setBool(
	fieldValue reflect.Value,
	envVal string,
) error {
	b, err := strconv.ParseBool(envVal)
	if err != nil {
		return fmt.Errorf("failed to parse bool: %w", err)
	}

	fieldValue.SetBool(b)

	return nil
}

func setDuration(
	fieldValue reflect.Value,
	envVal string,
) error {
	d, err := time.ParseDuration(envVal)
	if err != nil {
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	fieldValue.Set(reflect.ValueOf(d))

	return nil
}

func setStringSlice(
	fieldValue reflect.Value,
	envVal string,
) error {
	if envVal == "" {
		fieldValue.Set(reflect.ValueOf([]string{}))

		return nil
	}

	parts := strings.Split(envVal, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	fieldValue.Set(reflect.ValueOf(parts))

	return nil
}

func getDstStructValue(dst any) (reflect.Value, error) {
	if dst == nil {
		return reflect.Value{}, ErrNilDestination
	}

	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr {
		return reflect.Value{}, ErrTargetNotPointer
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return reflect.Value{}, ErrDestinationNotStruct
	}

	return val, nil
}

func isSupportedType(fieldValue reflect.Value) bool {
	// Handle time.Duration specifically
	if fieldValue.Type() == reflect.TypeFor[time.Duration]() {
		return true
	}

	// Handle []string specifically
	if fieldValue.Type() == reflect.TypeFor[[]string]() {
		return true
	}

	switch fieldValue.Kind() { //nolint:exhaustive
	case reflect.String,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Bool:
		return true
	default:
		return false
	}
}
