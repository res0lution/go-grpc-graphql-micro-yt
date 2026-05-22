package validator

import (
 "fmt"
 "strings"

 "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
 validate = validator.New()
}

func Validate(s any) error {
 if validate == nil {
  validate = validator.New()
 }

 return validate.Struct(s)
}

// Check slice length > 0
func HasLength[S ~[]T, T any](values S, field string) error {
 if len(values) == 0 {
  return fmt.Errorf("at least one %q is required", field)
 }

 return nil
}

// Check string is not empty with strings.TrimSpace
func IsEmptyString(s string) bool {
 return strings.EqualFold(strings.TrimSpace(s), "")
}

func NotEmptyString(s string, field string) error {
 if IsEmptyString(s) {
  return fmt.Errorf("string empty in %q", field)
 }

 return nil
}

type StructStub struct{}

type Set[T comparable] map[T]StructStub

func (s Set[T]) Has(key T) bool {
 _, yes := s[key]
 return yes
}

// Check slice contains only unique values
func ContainsUniqueValues[T comparable, S ~[]T](values S, field string) error {
 stub := StructStub{}
 set := make(Set[T], len(values))

 for i := range values {
  if set.Has(values[i]) {
   return fmt.Errorf("duplicate %q: %v", field, values[i])
  }

  set[values[i]] = stub
 }

 return nil
}

func NotContainsEmptyStrings(values []string, field string) error {
 for i, value := range values {
  if IsEmptyString(value) {
   return fmt.Errorf("%s at position %d is empty", field, i)
  }
 }

 return nil
}