package domain

import (
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
)

const (
	// TenantTextLen is the byte length of the text form that String and
	// MarshalText write.
	TenantTextLen = (uuidStrLen * 2) + 1 // 73 bytes

	uuidStrLen    = 36
	accountOffset = 0
	envOffset     = 16
)

var (
	ErrInvalidTenantTextLen = errors.New("tenant: text representation must be 73 characters")
	ErrInvalidTenantSep     = errors.New("tenant: missing '/' separator")
	ErrInvalidTenantBinLen  = errors.New("tenant: binary slice must be 32 bytes")
)

// Tenant represents an individual (account, env) combination, which is
// a natural point of isolation within our system.
type Tenant [32]byte

// NewTenant returns a Tenant for an AccountID and EnvID.  The format is an array
// for perf.
func NewTenant(account AccountID, env EnvID) Tenant {
	var t Tenant
	copy(t[accountOffset:envOffset], account[:])
	copy(t[envOffset:], env[:])
	return t
}

func (t *Tenant) AccountID() AccountID {
	return AccountID(t[accountOffset:envOffset])
}

func (t *Tenant) EnvID() EnvID {
	return EnvID(t[envOffset:])
}

// String returns the two UUIDs in dashed form joined by '/'.
func (t Tenant) String() string {
	var buf [TenantTextLen]byte
	t.encodeStringTo(&buf)
	return string(buf[:])
}

// AppendText implements encoding.TextAppender.
func (t Tenant) AppendText(b []byte) ([]byte, error) {
	var buf [TenantTextLen]byte
	t.encodeStringTo(&buf)
	return append(b, buf[:]...), nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (t Tenant) MarshalBinary() ([]byte, error) {
	b := make([]byte, len(t))
	copy(b, t[:])
	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (t *Tenant) UnmarshalBinary(data []byte) error {
	if len(data) != len(t) {
		return ErrInvalidTenantBinLen
	}
	copy(t[:], data)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (t Tenant) MarshalText() ([]byte, error) {
	b := make([]byte, TenantTextLen)
	t.encodeStringTo((*[TenantTextLen]byte)(b))
	return b, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.  it accepts only the
// exact form that MarshalText writes.
func (t *Tenant) UnmarshalText(text []byte) error {
	if len(text) != TenantTextLen {
		return ErrInvalidTenantTextLen
	}
	if text[36] != '/' {
		return ErrInvalidTenantSep
	}

	acc, err := uuid.ParseBytes(text[0:36])
	if err != nil {
		return err
	}
	env, err := uuid.ParseBytes(text[37:73])
	if err != nil {
		return err
	}

	*t = NewTenant(acc, env)
	return nil
}

func (t *Tenant) encodeStringTo(dst *[TenantTextLen]byte) {
	encodeUUID((*[uuidStrLen]byte)(dst[0:36]), t.AccountID())
	dst[36] = '/'
	encodeUUID((*[uuidStrLen]byte)(dst[37:73]), t.EnvID())
}

// encodeUUID writes the dashed hex form of u into dst without allocating.
func encodeUUID(dst *[uuidStrLen]byte, u uuid.UUID) {
	hex.Encode(dst[0:8], u[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], u[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], u[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], u[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], u[10:16])
}
