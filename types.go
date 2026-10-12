// Package apiquery compiles declared API query capabilities into immutable,
// transport-neutral plans. It validates declared capabilities, not records or SQL.
// Applications own authentication, authorization callbacks, execution, pagination
// and collaborator lifetimes. Inputs are borrowed during calls; compiled plans
// snapshot their retained data and expose defensive copies for concurrent readers.
package apiquery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// ValueType identifies the wire and comparison semantics of a typed value.
type ValueType string

const (
	// TypeString identifies string text; direct construction does not validate UTF-8.
	TypeString ValueType = "string"
	// TypeInt identifies canonical decimal int64 text.
	TypeInt ValueType = "int"
	// TypeUint identifies canonical decimal uint64 text.
	TypeUint ValueType = "uint"
	// TypeFloat identifies finite canonical float64 text.
	TypeFloat ValueType = "float"
	// TypeBool identifies true or false text.
	TypeBool ValueType = "bool"
	// TypeTime identifies UTC RFC3339Nano text.
	TypeTime ValueType = "time"
	// TypeBytes identifies unpadded URL-safe base64 bytes.
	TypeBytes ValueType = "bytes"
	// TypeNull identifies a null cursor position, not a declared field or predicate value.
	TypeNull ValueType = "null"
)

// Value is a closed, typed query value. It deliberately cannot hold arbitrary
// Go values or persistence expressions.
type Value struct {
	typeName ValueType
	text     string
}

// StringValue constructs a string value without checking UTF-8 or size. Compile
// checks text length, not UTF-8; invalid UTF-8 can be repaired by JSON encoding.
func StringValue(value string) Value { return Value{typeName: TypeString, text: value} }

// IntValue constructs a signed integer value.
func IntValue(value int64) Value {
	return Value{typeName: TypeInt, text: strconv.FormatInt(value, 10)}
}

// UintValue constructs an unsigned integer value.
func UintValue(value uint64) Value {
	return Value{typeName: TypeUint, text: strconv.FormatUint(value, 10)}
}

// FloatValue constructs a finite floating-point value. NaN and infinities return
// the invalid zero Value, which cannot pass typed predicate compilation.
func FloatValue(value float64) Value {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Value{}
	}
	return Value{typeName: TypeFloat, text: strconv.FormatFloat(value, 'g', -1, 64)}
}

// BoolValue constructs a boolean value.
func BoolValue(value bool) Value {
	return Value{typeName: TypeBool, text: strconv.FormatBool(value)}
}

// TimeValue formats UTC RFC3339Nano text without validating the year range.
// Out-of-range years need caller validation; Compile does not parse this text again.
func TimeValue(value time.Time) Value {
	return Value{typeName: TypeTime, text: value.UTC().Format(time.RFC3339Nano)}
}

// BytesValue immediately encodes bytes as unpadded URL-safe base64 without
// retaining the input slice; nil and empty input both encode as empty text.
func BytesValue(value []byte) Value {
	return Value{typeName: TypeBytes, text: base64.RawURLEncoding.EncodeToString(value)}
}

// NullValue constructs an explicit null cursor position. Null is not a valid
// declared field type, filter value, or equality constraint.
func NullValue() Value { return Value{typeName: TypeNull} }

// Type reports the value's closed type.
func (v Value) Type() ValueType { return v.typeName }

// String reports stored value text, without further validation. Protected values should
// not be
// placed in diagnostics by callers.
func (v Value) String() string { return v.text }

// MarshalJSON emits a deterministic typed representation.
func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  ValueType `json:"type"`
		Value string    `json:"value"`
	}{Type: v.typeName, Value: v.text})
}

// UnmarshalJSON validates a known type and canonical payload text, assigning v
// only on success. Member matching follows encoding/json: extra members and case
// aliases are accepted, duplicate members use the last value, and an omitted or
// null string payload becomes empty. These are existing wire limitations, not a
// strict object-schema guarantee. Errors may contain input details; do not expose
// them as sanitized diagnostics. A successful decode requires a non-nil receiver.
func (v *Value) UnmarshalJSON(data []byte) error {
	var encoded struct {
		Type  ValueType `json:"type"`
		Value string    `json:"value"`
	}
	if err := json.Unmarshal(data, &encoded); err != nil {
		return fmt.Errorf("decode typed value: %w", err)
	}
	if !validValueType(encoded.Type) || !canonicalValue(encoded.Type, encoded.Value) {
		return fmt.Errorf("decode typed value: invalid %s value", encoded.Type)
	}
	v.typeName = encoded.Type
	v.text = encoded.Value
	return nil
}

// canonicalValue checks payload text, not the surrounding JSON object shape.
func canonicalValue(valueType ValueType, value string) bool {
	switch valueType {
	case TypeString:
		return true
	case TypeInt:
		parsed, err := strconv.ParseInt(value, 10, 64)
		return err == nil && strconv.FormatInt(parsed, 10) == value
	case TypeUint:
		parsed, err := strconv.ParseUint(value, 10, 64)
		return err == nil && strconv.FormatUint(parsed, 10) == value
	case TypeFloat:
		parsed, err := strconv.ParseFloat(value, 64)
		return err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed) &&
			strconv.FormatFloat(parsed, 'g', -1, 64) == value
	case TypeBool:
		parsed, err := strconv.ParseBool(value)
		return err == nil && strconv.FormatBool(parsed) == value
	case TypeTime:
		parsed, err := time.Parse(time.RFC3339Nano, value)
		return err == nil && parsed.UTC().Format(time.RFC3339Nano) == value
	case TypeBytes:
		parsed, err := base64.RawURLEncoding.DecodeString(value)
		return err == nil && base64.RawURLEncoding.EncodeToString(parsed) == value
	case TypeNull:
		return value == ""
	default:
		return false
	}
}

func validValueType(valueType ValueType) bool {
	return validType(valueType) || valueType == TypeNull
}

// Optional distinguishes an absent request component from an explicitly
// supplied zero or empty value.
type Optional[T any] struct {
	present bool
	value   T
}

// Present constructs an explicitly supplied request component by value. It does
// not clone slices, maps or objects referenced by T.
func Present[T any](value T) Optional[T] { return Optional[T]{present: true, value: value} }

// IsPresent reports whether the component was supplied.
func (o Optional[T]) IsPresent() bool { return o.present }

// Value returns the supplied value and whether it was present. References inside
// T remain shared with the caller; the zero Optional is absent.
func (o Optional[T]) Value() (T, bool) { return o.value, o.present }

// Operator is a declared, typed predicate operation.
type Operator string

const (
	// OpEqual compares one value for equality.
	OpEqual Operator = "eq"
	// OpNotEqual compares one value for inequality.
	OpNotEqual Operator = "neq"
	// OpLess compares one ordered value with less-than.
	OpLess Operator = "lt"
	// OpLessOrEqual compares one ordered value with less-than-or-equal.
	OpLessOrEqual Operator = "lte"
	// OpGreater compares one ordered value with greater-than.
	OpGreater Operator = "gt"
	// OpGreaterOrEqual compares one ordered value with greater-than-or-equal.
	OpGreaterOrEqual Operator = "gte"
	// OpIn tests membership in a nonempty, duplicate-free value list.
	OpIn Operator = "in"
	// OpNotIn tests exclusion from a nonempty, duplicate-free value list.
	OpNotIn Operator = "not_in"
	// OpBetween uses two distinct ordered operands; Compile does not reorder or compare
	// endpoints.
	OpBetween Operator = "between"
	// OpIsNull tests nullness with no operands and requires a nullable declaration.
	OpIsNull Operator = "is_null"
	// OpContains matches literal string containment; the PostgreSQL adapter escapes LIKE
	// metacharacters.
	OpContains Operator = "contains"
	// OpStartsWith matches a literal string prefix.
	OpStartsWith Operator = "starts_with"
	// OpEndsWith matches a literal string suffix.
	OpEndsWith Operator = "ends_with"
)

// Logic identifies a bounded logical composition.
type Logic string

const (
	// LogicAnd requires every child predicate/group.
	LogicAnd Logic = "and"
	// LogicOr accepts an alternative child predicate/group.
	LogicOr Logic = "or"
	// LogicNot negates exactly one child.
	LogicNot Logic = "not"
)

// Predicate is a typed filter leaf.
type Predicate struct {
	// Name selects a declared filter capability.
	Name string `json:"name"`
	// Operator must be allowed for Name and determines operand arity.
	Operator Operator `json:"operator"`
	// Values contains typed operands; membership and between values must not repeat.
	Values []Value `json:"values"`
}

// FilterExpr is either one Predicate or one logical group.
type FilterExpr struct {
	// Predicate selects a leaf and must not coexist with Children.
	Predicate *Predicate `json:"predicate,omitempty"`
	// Logic selects a declared group operation; it is ignored on a predicate leaf.
	Logic Logic `json:"logic,omitempty"`
	// Children contains a nonempty group; LogicNot requires exactly one child.
	Children []FilterExpr `json:"children,omitempty"`
}

// Direction controls ordered sorting.
type Direction string

const (
	// Ascending requests ascending order.
	Ascending Direction = "asc"
	// Descending requests descending order.
	Descending Direction = "desc"
)

// NullOrder declares deterministic null placement.
type NullOrder string

const (
	// NullsFirst places nulls before non-null values.
	NullsFirst NullOrder = "first"
	// NullsLast places nulls after non-null values.
	NullsLast NullOrder = "last"
)

// SortTerm is one ordered sort component.
type SortTerm struct {
	// Name selects a declared sortable capability.
	Name string `json:"name"`
	// Direction must be Ascending or Descending.
	Direction Direction `json:"direction"`
	// Nulls may omit the schema null order but cannot override it with a different
	// explicit order.
	Nulls NullOrder `json:"nulls,omitempty"`
}

// PageMode selects a declared pagination capability.
type PageMode string

const (
	// PageNone disables pagination for this request.
	PageNone PageMode = "none"
	// PageCursor requests application-owned cursor seek pagination.
	PageCursor PageMode = "cursor"
	// PageOffset requests bounded offset pagination.
	PageOffset PageMode = "offset"
)

// CursorDirection identifies the seek direction authenticated by a cursor.
type CursorDirection string

const (
	// CursorForward seeks after a boundary.
	CursorForward CursorDirection = "forward"
	// CursorBackward seeks before a boundary.
	CursorBackward CursorDirection = "backward"
)

// CursorState is the authenticated, typed cursor state retained by a plan.
// Positions are sensitive application data and must not be logged.
type CursorState struct {
	// Direction is the decoded seek direction; Compile matches it to after/before.
	Direction CursorDirection `json:"direction"`
	// Positions contains one typed position per sort term; Compile snapshots this
	// sensitive slice.
	Positions []Value `json:"positions"`
	// Policy is authenticated application metadata; Compile checks size, not an expected
	// policy identity.
	Policy string `json:"policy,omitempty"`
}

// PageRequest describes bounded pagination without decoding a cursor.
type PageRequest struct {
	// Mode selects a declared mode; empty normalizes to PageNone.
	Mode PageMode `json:"mode"`
	// Size uses the schema default when zero in a paginated mode.
	Size int `json:"size,omitempty"`
	// After is a forward token, mutually exclusive with Before and erased after successful
	// decoding.
	After string `json:"after,omitempty"`
	// Before is a backward token, mutually exclusive with After and erased after
	// successful decoding.
	Before string `json:"before,omitempty"`
	// Offset is nonnegative within the declared offset ceiling and cannot accompany cursor
	// mode.
	Offset int `json:"offset,omitempty"`
}

// Request is the transport-neutral query input.
type Request struct {
	// SchemaRevision optionally requires an exact schema revision.
	SchemaRevision Optional[string]
	// Fields distinguishes absent defaults from an explicitly empty response fieldset.
	Fields Optional[[]string]
	// Includes contains requested dotted paths; absence selects none.
	Includes Optional[[]string]
	// Filter is a borrowed expression tree during Compile; nil selects no client filter.
	Filter *FilterExpr
	// Sorts distinguishes absent defaults from explicitly empty sorting; cursor mode still
	// adds its tie-breaker.
	Sorts Optional[[]SortTerm]
	// Page requests pagination without executing a backend query.
	Page PageRequest
}
