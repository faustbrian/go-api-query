// Package apiquerylegacylist parses the legacy JSON-RPC list profile that
// Laravel-era list methods expose: resource-keyed filters and sorts with long
// operator names and and/or booleans, plus Laravel CursorPaginator cursors and
// their self/prev/next links. It is a compatibility adapter beside the
// schema-compiled apiquery request, not a second query language for new
// methods.
//
// Parsing only checks names against the caller's Profile. A declared attribute
// is a capability the method can execute, not a permission: authorization,
// tenant constraints and value typing stay with the caller. Cursors are
// unsigned and unencrypted Base64url JSON, so decoded positions are untrusted
// client input and must never carry secrets or authorization state. Use the
// cursor package for authenticated cursors. All functions are stateless and
// safe for concurrent use; returned values never alias caller input.
package apiquerylegacylist

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

var (
	// ErrInvalid is wrapped by every *Error describing rejected parameters.
	ErrInvalid = errors.New("legacy list parameters are invalid")
	// ErrInvalidProfile reports a Profile that cannot parse anything.
	ErrInvalidProfile = errors.New("legacy list profile is invalid")
	// ErrInvalidCursor reports a cursor that cannot be decoded or encoded.
	ErrInvalidCursor = errors.New("legacy list cursor is invalid")
)

// Operator is a long legacy filter operator name.
type Operator string

// The fourteen operators shared by the legacy list methods.
const (
	// Equals matches an equal value.
	Equals Operator = "equals"
	// NotEquals matches a different value.
	NotEquals Operator = "not_equals"
	// GreaterThan matches a strictly greater value.
	GreaterThan Operator = "greater_than"
	// GreaterThanOrEqualTo matches a greater or equal value.
	GreaterThanOrEqualTo Operator = "greater_than_or_equal_to"
	// LessThan matches a strictly smaller value.
	LessThan Operator = "less_than"
	// LessThanOrEqualTo matches a smaller or equal value.
	LessThanOrEqualTo Operator = "less_than_or_equal_to"
	// Like matches a SQL LIKE pattern.
	Like Operator = "like"
	// NotLike excludes a SQL LIKE pattern.
	NotLike Operator = "not_like"
	// In matches any element of a JSON array value.
	In Operator = "in"
	// NotIn excludes every element of a JSON array value.
	NotIn Operator = "not_in"
	// Between matches the inclusive range of a two-element array value.
	Between Operator = "between"
	// NotBetween excludes the inclusive range of a two-element array value.
	NotBetween Operator = "not_between"
	// IsNull matches a null attribute and ignores the value.
	IsNull Operator = "is_null"
	// IsNotNull matches a non-null attribute and ignores the value.
	IsNotNull Operator = "is_not_null"
)

var operators = []Operator{
	Equals, NotEquals, GreaterThan, GreaterThanOrEqualTo, LessThan, LessThanOrEqualTo,
	Like, NotLike, In, NotIn, Between, NotBetween, IsNull, IsNotNull,
}

// Operators returns a fresh copy of every operator in declaration order.
func Operators() []Operator { return slices.Clone(operators) }

// ParseOperator matches a long operator name exactly and case-sensitively.
func ParseOperator(name string) (Operator, bool) {
	operator := Operator(name)
	return operator, slices.Contains(operators, operator)
}

// Boolean joins a filter to the filters before it.
type Boolean string

// Filter booleans. An absent, empty or null boolean is And.
const (
	// And requires the filter and the filters before it.
	And Boolean = "and"
	// Or accepts the filter or the filters before it.
	Or Boolean = "or"
)

// Direction is a sort direction.
type Direction string

// Sort directions; there is no default.
const (
	// Ascending sorts smallest first.
	Ascending Direction = "asc"
	// Descending sorts largest first.
	Descending Direction = "desc"
)

// Resource declares the attribute names a method accepts under one resource
// key, such as "self" or a related resource.
type Resource struct {
	// Name is the resource key, unique and non-empty within its list.
	Name string
	// Attributes are the accepted names, unique and non-empty. An empty list
	// accepts the key but rejects every entry.
	Attributes []string
}

// Profile declares what one list method accepts. Resources are evaluated in
// declaration order and Parse functions borrow the profile for the call.
type Profile struct {
	// Filters lists the resources and attributes ParseFilters accepts.
	Filters []Resource
	// Sorts lists the resources and attributes ParseSorts accepts.
	Sorts []Resource
	// IgnoreUndeclaredResources skips undeclared resource keys without decoding
	// them instead of rejecting them with KindResource. The zero value rejects.
	IgnoreUndeclaredResources bool
	// MaxBytes bounds the raw filters or sorts member before decoding and must
	// be positive.
	MaxBytes int
}

// Filter is one parsed filter. Value holds the exact JSON value bytes, or nil
// when the member was absent; it is not type-checked against the attribute.
type Filter struct {
	// Resource is the declared resource key the filter appeared under.
	Resource string
	// Attribute is a declared attribute of Resource.
	Attribute string
	// Operator is one of the fourteen long operator names.
	Operator Operator
	// Value is the exact JSON value, or nil when absent.
	Value json.RawMessage
	// Boolean is And or Or; parsing normalizes an absent boolean to And.
	Boolean Boolean
}

// Sort is one parsed sort term.
type Sort struct {
	// Resource is the declared resource key the sort appeared under.
	Resource string
	// Attribute is a declared attribute of Resource.
	Attribute string
	// Direction is Ascending or Descending.
	Direction Direction
}

// ErrorKind classifies a rejected parameter so callers can keep their own
// legacy error documents.
type ErrorKind uint8

const (
	// KindShape is malformed JSON or a member of the wrong JSON type.
	KindShape ErrorKind = iota + 1
	// KindResource is an undeclared resource key.
	KindResource
	// KindAttribute is an undeclared attribute name.
	KindAttribute
	// KindOperator is an unknown operator name.
	KindOperator
	// KindBoolean is a boolean other than and or or.
	KindBoolean
	// KindDirection is a sort direction other than asc or desc.
	KindDirection
	// KindValue is a filter value of the wrong arity for its operator.
	KindValue
)

// Error describes one rejected parameter. Pointer is a JSON pointer such as
// "/params/filters" or "/params/sorts/self". Unknown is the client-supplied
// name or keyword that was rejected and Allowed the declared alternatives for
// KindResource and KindAttribute. Error() never includes Unknown.
type Error struct {
	// Kind classifies the failure.
	Kind ErrorKind
	// Pointer locates the rejected member.
	Pointer string
	// Unknown is the rejected client-supplied name or keyword, if any.
	Unknown string
	// Allowed lists the declared alternatives for KindResource and KindAttribute.
	Allowed []string
}

// Error returns fixed text naming the kind and pointer, never client input.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s at %s", ErrInvalid, e.Kind, e.Pointer)
}

// Unwrap returns ErrInvalid.
func (e *Error) Unwrap() error { return ErrInvalid }

// String returns a short fixed description of the kind.
func (k ErrorKind) String() string {
	switch k {
	case KindShape:
		return "malformed member"
	case KindResource:
		return "undeclared resource"
	case KindAttribute:
		return "undeclared attribute"
	case KindOperator:
		return "unknown operator"
	case KindBoolean:
		return "unknown boolean"
	case KindDirection:
		return "unknown direction"
	case KindValue:
		return "invalid value"
	default:
		return "unknown failure"
	}
}

const (
	filtersPointer = "/params/filters"
	sortsPointer   = "/params/sorts"
)

type rawFilter struct {
	Attribute string          `json:"attribute"`
	Operator  string          `json:"operator"`
	Value     json.RawMessage `json:"value"`
	Boolean   string          `json:"boolean"`
}

type rawSort struct {
	Attribute string `json:"attribute"`
	Direction string `json:"direction"`
}

// ParseFilters decodes {"<resource>": [{"attribute", "operator", "value",
// "boolean"}]}. Absent, whitespace-only or null input and null resource arrays
// yield no filters. Declared resources are read in profile order, each entry
// checked for attribute, then operator, then boolean; undeclared resources are
// reported afterwards, the alphabetically first one winning. Member names
// match case-insensitively and unknown members are ignored, as encoding/json
// does. Value arity is checked separately by Filter.ValidateValue.
func (p Profile) ParseFilters(raw json.RawMessage) ([]Filter, error) {
	resources, err := p.decodeResources(raw, filtersPointer)
	if err != nil || resources == nil {
		return nil, err
	}
	var filters []Filter
	for _, resource := range p.Filters {
		member, present := resources[resource.Name]
		if !present {
			continue
		}
		var entries []rawFilter
		if json.Unmarshal(member, &entries) != nil {
			return nil, &Error{Kind: KindShape, Pointer: filtersPointer + "/" + resource.Name}
		}
		for _, entry := range entries {
			if !slices.Contains(resource.Attributes, entry.Attribute) {
				return nil, &Error{Kind: KindAttribute, Pointer: filtersPointer, Unknown: entry.Attribute,
					Allowed: slices.Clone(resource.Attributes)}
			}
			operator, ok := ParseOperator(entry.Operator)
			if !ok {
				return nil, &Error{Kind: KindOperator, Pointer: filtersPointer, Unknown: entry.Operator}
			}
			boolean := Boolean(entry.Boolean)
			if boolean == "" {
				boolean = And
			}
			if boolean != And && boolean != Or {
				return nil, &Error{Kind: KindBoolean, Pointer: filtersPointer, Unknown: entry.Boolean}
			}
			filters = append(filters, Filter{Resource: resource.Name, Attribute: entry.Attribute,
				Operator: operator, Value: bytes.Clone(entry.Value), Boolean: boolean})
		}
	}
	if err := p.rejectUndeclared(resources, p.Filters, filtersPointer); err != nil {
		return nil, err
	}
	return filters, nil
}

// ParseSorts decodes {"<resource>": [{"attribute", "direction"}]}. Absent or
// null input yields no sorts. Unlike filters, undeclared resources are
// reported before any declared entry is checked; entries are then checked for
// attribute and direction in profile order.
func (p Profile) ParseSorts(raw json.RawMessage) ([]Sort, error) {
	resources, err := p.decodeResources(raw, sortsPointer)
	if err != nil || resources == nil {
		return nil, err
	}
	if err := p.rejectUndeclared(resources, p.Sorts, sortsPointer); err != nil {
		return nil, err
	}
	var sorts []Sort
	for _, resource := range p.Sorts {
		member, present := resources[resource.Name]
		if !present {
			continue
		}
		var entries []rawSort
		if json.Unmarshal(member, &entries) != nil {
			return nil, &Error{Kind: KindShape, Pointer: sortsPointer + "/" + resource.Name}
		}
		for _, entry := range entries {
			if !slices.Contains(resource.Attributes, entry.Attribute) {
				return nil, &Error{Kind: KindAttribute, Pointer: sortsPointer, Unknown: entry.Attribute,
					Allowed: slices.Clone(resource.Attributes)}
			}
			direction := Direction(entry.Direction)
			if direction != Ascending && direction != Descending {
				return nil, &Error{Kind: KindDirection, Pointer: sortsPointer, Unknown: entry.Direction}
			}
			sorts = append(sorts, Sort{Resource: resource.Name, Attribute: entry.Attribute, Direction: direction})
		}
	}
	return sorts, nil
}

// decodeResources returns nil without error for absent or null input.
func (p Profile) decodeResources(raw json.RawMessage, pointer string) (map[string]json.RawMessage, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	if len(raw) > p.MaxBytes {
		return nil, &Error{Kind: KindShape, Pointer: pointer}
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var resources map[string]json.RawMessage
	if json.Unmarshal(trimmed, &resources) != nil {
		return nil, &Error{Kind: KindShape, Pointer: pointer}
	}
	return resources, nil
}

func (p Profile) rejectUndeclared(resources map[string]json.RawMessage, declared []Resource, pointer string) error {
	if p.IgnoreUndeclaredResources {
		return nil
	}
	var undeclared []string
	for name := range resources {
		if !slices.ContainsFunc(declared, func(resource Resource) bool { return resource.Name == name }) {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	names := make([]string, len(declared))
	for index, resource := range declared {
		names[index] = resource.Name
	}
	return &Error{Kind: KindResource, Pointer: pointer, Unknown: slices.Min(undeclared), Allowed: names}
}

func (p Profile) validate() error {
	if p.MaxBytes <= 0 {
		return fmt.Errorf("%w: MaxBytes must be positive", ErrInvalidProfile)
	}
	for _, declared := range [][]Resource{p.Filters, p.Sorts} {
		for index, resource := range declared {
			if resource.Name == "" || slices.ContainsFunc(declared[:index], func(other Resource) bool { return other.Name == resource.Name }) {
				return fmt.Errorf("%w: resource names must be unique and non-empty", ErrInvalidProfile)
			}
			for position, attribute := range resource.Attributes {
				if attribute == "" || slices.Contains(resource.Attributes[:position], attribute) {
					return fmt.Errorf("%w: attribute names must be unique and non-empty", ErrInvalidProfile)
				}
			}
		}
	}
	return nil
}

// ValidateValue checks value arity: in and not_in need a JSON array, between
// and not_between an array of exactly two elements, and other operators accept
// any value including an absent one. It returns a KindValue *Error.
func (f Filter) ValidateValue() error {
	value := bytes.TrimSpace(f.Value)
	if len(value) != 0 && !json.Valid(value) {
		return &Error{Kind: KindValue, Pointer: filtersPointer, Unknown: f.Attribute}
	}
	var elements []json.RawMessage
	switch f.Operator {
	case In, NotIn:
		if len(value) == 0 || value[0] != '[' || json.Unmarshal(value, &elements) != nil {
			return &Error{Kind: KindValue, Pointer: filtersPointer, Unknown: f.Attribute}
		}
	case Between, NotBetween:
		if len(value) == 0 || value[0] != '[' || json.Unmarshal(value, &elements) != nil || len(elements) != 2 {
			return &Error{Kind: KindValue, Pointer: filtersPointer, Unknown: f.Attribute}
		}
	case Equals, NotEquals, GreaterThan, GreaterThanOrEqualTo, LessThan, LessThanOrEqualTo,
		Like, NotLike, IsNull, IsNotNull:
	}
	return nil
}

// PointsToNextItemsKey is the reserved cursor member holding the direction.
const PointsToNextItemsKey = "_pointsToNextItems"

// CursorParameter is one named boundary position. Value is exact JSON.
type CursorParameter struct {
	// Name is the sort attribute or column name, unique within a cursor.
	Name string
	// Value is the exact JSON value of the boundary item.
	Value json.RawMessage
}

// Cursor is a Laravel CursorPaginator position: the boundary item's sort
// values and whether the cursor points at the items after it.
type Cursor struct {
	// Parameters are the boundary positions in document order.
	Parameters []CursorParameter
	// PointsToNextItems is true for a forward cursor and false for a backward one.
	PointsToNextItems bool
}

// EncodeCursor writes the parameters in order, then PointsToNextItemsKey, as
// one compact JSON object in unpadded Base64url. Values are compacted with
// encoding/json escaping. It needs at least one parameter with unique,
// non-empty, non-reserved names and valid JSON values.
func EncodeCursor(cursor Cursor) (string, error) {
	if len(cursor.Parameters) == 0 {
		return "", ErrInvalidCursor
	}
	var document bytes.Buffer
	document.WriteByte('{')
	for index, parameter := range cursor.Parameters {
		if parameter.Name == "" || parameter.Name == PointsToNextItemsKey ||
			slices.ContainsFunc(cursor.Parameters[:index], func(other CursorParameter) bool { return other.Name == parameter.Name }) {
			return "", ErrInvalidCursor
		}
		name, _ := json.Marshal(parameter.Name)
		value, err := json.Marshal(parameter.Value)
		if err != nil || len(parameter.Value) == 0 {
			return "", ErrInvalidCursor
		}
		document.Write(name)
		document.WriteByte(':')
		document.Write(value)
		document.WriteByte(',')
	}
	fmt.Fprintf(&document, `"%s":%t}`, PointsToNextItemsKey, cursor.PointsToNextItems)
	return base64.RawURLEncoding.EncodeToString(document.Bytes()), nil
}

// DecodeCursor accepts an unpadded Base64url JSON object of at most maxBytes
// encoded bytes with exactly one boolean PointsToNextItemsKey, at least one
// other member, unique non-empty names, and nothing but whitespace after it.
// Parameters keep document order and exact value bytes. It returns ErrInvalidCursor
// and the zero Cursor on any failure. The cursor is not authenticated.
func DecodeCursor(encoded string, maxBytes int) (Cursor, error) {
	if encoded == "" || len(encoded) > maxBytes {
		return Cursor{}, ErrInvalidCursor
	}
	document, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	cursor, ok := decodeCursorDocument(document)
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	return cursor, nil
}

func decodeCursorDocument(document []byte) (Cursor, bool) {
	decoder := json.NewDecoder(bytes.NewReader(document))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return Cursor{}, false
	}
	var cursor Cursor
	direction := false
	for decoder.More() {
		token, err := decoder.Token()
		name, ok := token.(string)
		if err != nil || !ok || name == "" {
			return Cursor{}, false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return Cursor{}, false
		}
		if name == PointsToNextItemsKey {
			if direction || json.Unmarshal(value, &cursor.PointsToNextItems) != nil ||
				(!bytes.Equal(value, []byte("true")) && !bytes.Equal(value, []byte("false"))) {
				return Cursor{}, false
			}
			direction = true
			continue
		}
		if slices.ContainsFunc(cursor.Parameters, func(other CursorParameter) bool { return other.Name == name }) {
			return Cursor{}, false
		}
		cursor.Parameters = append(cursor.Parameters, CursorParameter{Name: name, Value: value})
	}
	if _, err := decoder.Token(); err != nil || !direction || len(cursor.Parameters) == 0 {
		return Cursor{}, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Cursor{}, false
	}
	return cursor, true
}

// Values returns the parameter values in the order of names when the cursor
// holds exactly those parameters, as a cursor issued for the requested sort
// does; otherwise it returns nil and false.
func (c Cursor) Values(names ...string) ([]json.RawMessage, bool) {
	if len(names) == 0 || len(names) != len(c.Parameters) {
		return nil, false
	}
	values := make([]json.RawMessage, len(names))
	for index, name := range names {
		position := slices.IndexFunc(c.Parameters, func(parameter CursorParameter) bool { return parameter.Name == name })
		if position < 0 || slices.Contains(names[:index], name) {
			return nil, false
		}
		values[index] = bytes.Clone(c.Parameters[position].Value)
	}
	return values, true
}

// Page describes one served page for link generation.
type Page struct {
	// Current is the decoded request cursor, nil for an initial request.
	Current *Cursor
	// First holds the cursor parameters of the first returned item, empty when
	// the page has no items.
	First []CursorParameter
	// Last holds the cursor parameters of the last returned item, empty when
	// the page has no items.
	Last []CursorParameter
	// HasMore reports another item beyond the page in the traversal direction.
	HasMore bool
}

// Links holds encoded cursors; an empty string means the link is absent.
// It marshals as {"self", "prev", "next"} with absent links omitted.
type Links struct {
	// Self is the re-encoded request cursor.
	Self string `json:"self,omitempty"`
	// Prev points backward from the first item.
	Prev string `json:"prev,omitempty"`
	// Next points forward from the last item.
	Next string `json:"next,omitempty"`
}

// Empty reports that no link exists, in which case legacy methods omit the
// page metadata entirely.
func (l Links) Empty() bool { return l == Links{} }

// Links applies Laravel CursorPaginator's rules. Self re-encodes Current. With
// items, Prev points backward from First when there is a Current cursor that
// points forward or more items remain, and Next points forward from Last when
// more items remain or Current points backward. A page without items has only
// Self. Invalid positions return ErrInvalidCursor and empty Links.
func (p Page) Links() (Links, error) {
	var links Links
	var err error
	if p.Current != nil {
		if links.Self, err = EncodeCursor(*p.Current); err != nil {
			return Links{}, err
		}
	}
	if len(p.First) == 0 && len(p.Last) == 0 {
		if p.HasMore {
			return Links{}, ErrInvalidCursor
		}
		return links, nil
	}
	if len(p.First) == 0 || len(p.Last) == 0 {
		return Links{}, ErrInvalidCursor
	}
	if p.Current != nil && (p.Current.PointsToNextItems || p.HasMore) {
		if links.Prev, err = EncodeCursor(Cursor{Parameters: p.First}); err != nil {
			return Links{}, err
		}
	}
	if p.HasMore || (p.Current != nil && !p.Current.PointsToNextItems) {
		if links.Next, err = EncodeCursor(Cursor{Parameters: p.Last, PointsToNextItems: true}); err != nil {
			return Links{}, err
		}
	}
	return links, nil
}
