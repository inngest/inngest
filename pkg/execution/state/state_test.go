package state

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestCustomConcurrency_ParseKey(t *testing.T) {
	r := require.New(t)
	someUUID := uuid.New()
	someHash := strconv.FormatUint(xxhash.Sum64String("abcdef0123456789"), 36)

	t.Run("function scope", func(t *testing.T) {
		sut := CustomConcurrency{
			Key:   fmt.Sprintf("f:%s:%s", someUUID.String(), someHash),
			Hash:  someHash,
			Limit: 20,
		}

		scope, id, hash, err := sut.ParseKey()
		r.NoError(err)
		r.Equal(enums.ConcurrencyScopeFn, scope)
		r.Equal(someUUID, id)
		r.Equal(someHash, hash)
	})

	t.Run("environment scope", func(t *testing.T) {
		sut := CustomConcurrency{
			Key:   fmt.Sprintf("e:%s:%s", someUUID.String(), someHash),
			Hash:  someHash,
			Limit: 20,
		}

		scope, id, hash, err := sut.ParseKey()
		r.NoError(err)
		r.Equal(enums.ConcurrencyScopeEnv, scope)
		r.Equal(someUUID, id)
		r.Equal(someHash, hash)
	})
}

func TestCustomConcurrency_Validate(t *testing.T) {
	r := require.New(t)

	cases := []struct {
		name    string
		cc      CustomConcurrency
		wantErr bool
	}{
		{
			name: "happy path (fn scope)",
			cc: CustomConcurrency{
				Key:   "f:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:event.data.user",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: false,
		},
		{
			name: "happy path (env scope)",
			cc: CustomConcurrency{
				Key:   "e:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:event.data.user",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: false,
		},
		{
			name: "happy path (account scope)",
			cc: CustomConcurrency{
				Key:   "a:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:event.data.user",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: false,
		},
		{
			name: "invalid scope",
			cc: CustomConcurrency{
				Key:   "x:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:event.data.user",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: true,
		},
		{
			name: "impossibly short",
			cc: CustomConcurrency{
				Key:   "a:x:",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: true,
		},
		{
			name: "too few parts",
			cc: CustomConcurrency{
				Key:   "f:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: true,
		},
		{
			name: "too many parts",
			cc: CustomConcurrency{
				Key:   "f:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:evt.data.user:extra",
				Hash:  "abcd1234",
				Limit: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid limit",
			cc: CustomConcurrency{
				Key:   "f:AA280D9C-AD14-44FD-AA94-A8A0D8D9D876:event.data.user",
				Hash:  "abcd1234",
				Limit: -1,
			},
			wantErr: true,
		},
	}
	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			r.Equal(tCase.wantErr, tCase.cc.Validate() != nil)
		})
	}
}
