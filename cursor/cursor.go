// Package cursor encrypts, authenticates, versions, and bounds typed cursor
// positions without coupling them to a transport or database.
package cursor

import (
	"bytes"
	"cmp"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	apiquery "github.com/faustbrian/go-api-query/v5"
)

var (
	// ErrInvalid reports a malformed, oversized, unknown-key, or tampered cursor.
	ErrInvalid = errors.New("cursor is invalid")
	// ErrExpired reports a cursor whose configured lifetime has ended.
	ErrExpired = errors.New("cursor has expired")
	// ErrVersion reports an unsupported cursor protocol version.
	ErrVersion = errors.New("cursor version is not supported")
	// ErrSchema reports a cursor compiled for another schema revision.
	ErrSchema = errors.New("cursor schema does not match")
	// ErrSort reports a cursor compiled for another ordered sort definition.
	ErrSort = errors.New("cursor sort does not match")
	// ErrReplay reports a valid cursor already consumed under replay policy.
	ErrReplay = errors.New("cursor replay is not allowed")
)

// Direction identifies forward or backward traversal.
type Direction = apiquery.CursorDirection

const (
	// Forward is the forward cursor direction.
	Forward Direction = apiquery.CursorForward
	// Backward is the backward cursor direction.
	Backward Direction = apiquery.CursorBackward
)

// Payload binds typed positions to schema, sort, direction, expiry, and policy.
type Payload struct {
	// SchemaRevision binds the cursor to one application schema revision.
	SchemaRevision string
	// Direction selects forward or backward traversal.
	Direction Direction
	// Sorts is the exact ordered sort contract; Encode borrows it for the call.
	Sorts []apiquery.SortTerm
	// Positions contains sensitive typed boundary values in sort order; Encode borrows it
	// for the call.
	Positions []apiquery.Value
	// ExpiresAt must be after the configured clock and no more than MaxTTL ahead.
	ExpiresAt time.Time
	// Policy is authenticated metadata; the caller owns matching it to current application
	// policy.
	Policy string
}

// Key is one named 256-bit AES key. Secret is defensively copied.
type Key struct {
	// ID is public nonsecret routing metadata, one to 64 ASCII letters, digits,
	// underscores or hyphens.
	ID string
	// Secret must contain 32 bytes and is copied by keyring construction/rotation; caller
	// custody remains its own.
	Secret []byte
}

// Keyring atomically rotates one active key and optional retained decode keys.
type Keyring struct {
	mu     sync.RWMutex
	active string
	keys   map[string][]byte
}

// NewKeyring constructs a keyring with one active key and retained keys.
func NewKeyring(active Key, retained ...Key) (*Keyring, error) {
	keyring := &Keyring{}
	if err := keyring.Rotate(active, retained...); err != nil {
		return nil, err
	}
	return keyring, nil
}

// Rotate validates and copies all keys before atomically replacing the snapshot.
// Failure leaves the old snapshot unchanged. An in-flight operation may finish
// using a key already copied before retirement; secrets are not explicitly zeroized.
func (k *Keyring) Rotate(active Key, retained ...Key) error {
	keys := make(map[string][]byte)
	all := append([]Key{active}, retained...)
	for _, key := range all {
		if !validKeyID(key.ID) || len(key.Secret) != 32 {
			return ErrInvalid
		}
		if _, duplicate := keys[key.ID]; duplicate {
			return ErrInvalid
		}
		keys[key.ID] = append([]byte(nil), key.Secret...)
	}
	k.mu.Lock()
	k.active = active.ID
	k.keys = keys
	k.mu.Unlock()
	return nil
}

// activeKey copies under the lock, so encryption never holds it across entropy
// reads or application callbacks. Retirement cannot revoke this copied snapshot.
func (k *Keyring) activeKey() (string, []byte, bool) {
	if k == nil {
		return "", nil, false
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	secret, exists := k.keys[k.active]
	return k.active, append([]byte(nil), secret...), exists
}

func (k *Keyring) key(id string) ([]byte, bool) {
	if k == nil {
		return nil, false
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	secret, exists := k.keys[id]
	return append([]byte(nil), secret...), exists
}

// Config contains hard cursor protocol limits and dependencies.
type Config struct {
	// Version is the caller-selected protocol identity authenticated in the envelope.
	Version string
	// Keys is a borrowed rotating keyring which must have an active key at construction.
	Keys *Keyring
	// MaxEncodedBytes bounds incoming tokens before decoding and outgoing plaintext/token
	// after encoding.
	MaxEncodedBytes int
	// MaxPositions is the positive maximum number of cursor positions.
	MaxPositions int
	// MaxStringBytes caps revision, policy, sort names and position text; nonpositive
	// values default to 256.
	MaxStringBytes int
	// MaxTTL is the positive maximum remaining lifetime accepted by Encode and Decode.
	MaxTTL time.Duration
	// Clock is a borrowed synchronous clock; nil defaults to time.Now and concurrent
	// callers may invoke it concurrently.
	Clock func() time.Time
	// ReplayStore is optional borrowed atomic-consumption storage; nil disables replay
	// prevention.
	ReplayStore ReplayStore
}

// ReplayStore atomically accepts a new opaque fingerprint or rejects replay.
// It must honor context cancellation, permit concurrent calls, and own bounded
// retention until expiry.
type ReplayStore func(ctx context.Context, fingerprint [32]byte, expiresAt time.Time) bool

// Codec encrypts and validates cursor payloads. It borrows the keyring, clock
// and optional replay store; keep them valid for its lifetime. Concurrent calls
// can invoke callbacks concurrently. No collaborator shutdown, panic recovery
// or background goroutine is supplied.
type Codec struct {
	version         string
	keys            *Keyring
	maxEncodedBytes int
	maxPositions    int
	maxStringBytes  int
	maxTTL          time.Duration
	clockMu         sync.RWMutex
	clock           func() time.Time
	replayStore     ReplayStore
	// entropy is fixed to crypto/rand at construction. Package tests replace it
	// only on an isolated codec to prove fail-closed reads without global state.
	entropy io.Reader
}

// NewCodec validates a bounded cursor protocol configuration.
func NewCodec(config Config) (*Codec, error) {
	if !validKeyID(config.Version) || config.Keys == nil || config.MaxEncodedBytes <= 0 ||
		config.MaxPositions <= 0 || config.MaxTTL <= 0 {
		return nil, ErrInvalid
	}
	if _, _, exists := config.Keys.activeKey(); !exists {
		return nil, ErrInvalid
	}
	if config.MaxStringBytes <= 0 {
		config.MaxStringBytes = 256
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	return &Codec{version: config.Version, keys: config.Keys,
		maxEncodedBytes: config.MaxEncodedBytes, maxPositions: config.MaxPositions,
		maxStringBytes: config.MaxStringBytes, maxTTL: config.MaxTTL,
		clock: config.Clock, replayStore: config.ReplayStore, entropy: rand.Reader}, nil
}

// SetClock replaces the time source atomically. It is intended for controlled
// tests and applications with an injected clock. nil leaves the existing clock
// unchanged. The borrowed callback executes outside the lock and must support
// concurrent calls when the codec is shared.
func (c *Codec) SetClock(clock func() time.Time) {
	if clock == nil {
		return
	}
	c.clockMu.Lock()
	c.clock = clock
	c.clockMu.Unlock()
}

// Encode returns an encrypted and authenticated opaque cursor using AES-256-GCM
// and library-selected crypto/rand entropy. It borrows payload slices during the
// synchronous call, checks size after allocations, and returns empty token on
// failure. Position payload text is not independently revalidated here.
func (c *Codec) Encode(payload Payload) (string, error) {
	now := c.now()
	if !c.validPayload(payload, now) {
		return "", ErrInvalid
	}
	keyID, secret, exists := c.keys.activeKey()
	if !exists {
		return "", ErrInvalid
	}
	wire := wirePayload{Version: c.version, SchemaRevision: payload.SchemaRevision,
		Direction: payload.Direction, Sorts: append([]apiquery.SortTerm(nil), payload.Sorts...),
		Positions: append([]apiquery.Value(nil), payload.Positions...),
		ExpiresAt: payload.ExpiresAt.UTC(), Policy: payload.Policy}
	plain, err := json.Marshal(wire)
	if err != nil {
		return "", ErrInvalid
	}
	if cmp.Compare(len(plain), c.maxEncodedBytes) == 1 {
		return "", ErrInvalid
	}
	aead, err := newAEAD(secret)
	if err != nil {
		return "", ErrInvalid
	}
	nonce := make([]byte, aead.NonceSize())
	_, randomErr := io.ReadFull(c.entropy, nonce)
	if randomErr != nil {
		return "", ErrInvalid
	}
	aad := c.version + "." + keyID
	sealed := aead.Seal(nonce, nonce, plain, []byte(aad))
	token := aad + "." + base64.RawURLEncoding.EncodeToString(sealed)
	if len(token) > c.maxEncodedBytes {
		return "", ErrInvalid
	}
	return token, nil
}

// Decode authenticates, decrypts, and binds a cursor to the expected schema
// revision and exact ordered sort definition. It rejects replay-enabled codecs;
// use DecodeContext for those, rather than hiding the caller context.
func (c *Codec) Decode(token, expectedSchema string, expectedSorts []apiquery.SortTerm) (Payload, error) {
	if c.replayStore != nil {
		return Payload{}, ErrInvalid
	}
	return c.DecodeContext(context.Background(), token, expectedSchema, expectedSorts)
}

// DecodeContext authenticates, decrypts, and binds a cursor while propagating
// cancellation to configured replay storage. ctx must be non-nil. Cancellation
// is checked at entry and before the store, not continuously; successful store
// acceptance is the commit point even if cancellation races with its return.
// Store calls are synchronous and outside library locks, with no forced preemption.
// Returned slices are caller-owned. Application policy identity must be compared
// separately; expiry/schema/sort checks do not establish application authorization.
func (c *Codec) DecodeContext(ctx context.Context, token, expectedSchema string,
	expectedSorts []apiquery.SortTerm) (Payload, error) {
	if ctx == nil {
		return Payload{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Payload{}, err
	}
	if token == "" {
		return Payload{}, ErrInvalid
	}
	if cmp.Compare(len(token), c.maxEncodedBytes) == 1 {
		return Payload{}, ErrInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Payload{}, ErrInvalid
	}
	if parts[0] != c.version {
		return Payload{}, ErrVersion
	}
	secret, exists := c.keys.key(parts[1])
	if !exists {
		return Payload{}, ErrInvalid
	}
	sealed, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Payload{}, ErrInvalid
	}
	aead, err := newAEAD(secret)
	if err != nil {
		return Payload{}, ErrInvalid
	}
	if cmp.Compare(len(sealed), aead.NonceSize()+aead.Overhead()) == -1 {
		return Payload{}, ErrInvalid
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(parts[0]+"."+parts[1]))
	if err != nil {
		return Payload{}, ErrInvalid
	}
	var wire wirePayload
	decoder := json.NewDecoder(bytes.NewReader(plain))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return Payload{}, ErrInvalid
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Payload{}, ErrInvalid
	}
	payload := Payload{SchemaRevision: wire.SchemaRevision, Direction: wire.Direction,
		Sorts: wire.Sorts, Positions: wire.Positions, ExpiresAt: wire.ExpiresAt,
		Policy: wire.Policy}
	now := c.now()
	if wire.Version != c.version || !c.validPayload(payload, now) {
		if !wire.ExpiresAt.After(now) {
			return Payload{}, ErrExpired
		}
		return Payload{}, ErrInvalid
	}
	if subtle.ConstantTimeCompare([]byte(wire.SchemaRevision), []byte(expectedSchema)) != 1 {
		return Payload{}, ErrSchema
	}
	if !equalSorts(wire.Sorts, expectedSorts) {
		return Payload{}, ErrSort
	}
	if c.replayStore != nil {
		if err := ctx.Err(); err != nil {
			return Payload{}, err
		}
		fingerprint := sha256.Sum256([]byte(token))
		accepted := c.replayStore(ctx, fingerprint, wire.ExpiresAt)
		if !accepted {
			if err := ctx.Err(); err != nil {
				return Payload{}, err
			}
			return Payload{}, ErrReplay
		}
	}
	return payload, nil
}

// DecodeCursor implements apiquery.CursorDecoder using this authenticated
// cursor protocol.
func (c *Codec) DecodeCursor(ctx context.Context, token, expectedSchema string,
	expectedSorts []apiquery.SortTerm) (apiquery.CursorState, error) {
	payload, err := c.DecodeContext(ctx, token, expectedSchema, expectedSorts)
	if err != nil {
		return apiquery.CursorState{}, err
	}
	return apiquery.CursorState{Direction: payload.Direction,
		Positions: append([]apiquery.Value(nil), payload.Positions...), Policy: payload.Policy}, nil
}

type wirePayload struct {
	Version        string              `json:"version"`
	SchemaRevision string              `json:"schema_revision"`
	Direction      Direction           `json:"direction"`
	Sorts          []apiquery.SortTerm `json:"sorts"`
	Positions      []apiquery.Value    `json:"positions"`
	ExpiresAt      time.Time           `json:"expires_at"`
	Policy         string              `json:"policy,omitempty"`
}

func (c *Codec) validPayload(payload Payload, now time.Time) bool {
	if len(payload.SchemaRevision) == 0 || len(payload.SchemaRevision) > c.maxStringBytes ||
		len(payload.Policy) > c.maxStringBytes || len(payload.Positions) == 0 ||
		len(payload.Positions) > c.maxPositions || len(payload.Positions) != len(payload.Sorts) ||
		!payload.ExpiresAt.After(now) || payload.ExpiresAt.Sub(now) > c.maxTTL ||
		(payload.Direction != Forward && payload.Direction != Backward) {
		return false
	}
	for index, sort := range payload.Sorts {
		if len(sort.Name) == 0 || len(sort.Name) > c.maxStringBytes ||
			(sort.Direction != apiquery.Ascending && sort.Direction != apiquery.Descending) ||
			(sort.Nulls != "" && sort.Nulls != apiquery.NullsFirst && sort.Nulls != apiquery.NullsLast) ||
			len(payload.Positions[index].String()) > c.maxStringBytes || payload.Positions[index].Type() == "" {
			return false
		}
	}
	return true
}

func (c *Codec) now() time.Time {
	c.clockMu.RLock()
	clock := c.clock
	c.clockMu.RUnlock()
	return clock().UTC()
}

func newAEAD(secret []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalid
	}
	return nil
}

func equalSorts(left, right []apiquery.SortTerm) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validKeyID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
