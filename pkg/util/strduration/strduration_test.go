package strduration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStringDurationMarshalJSON(t *testing.T) {
	sd := Duration(24*time.Hour + 30*time.Minute)

	data, err := json.Marshal(sd)
	require.NoError(t, err)
	require.Equal(t, `"1d30m"`, string(data))
}

func TestStringDurationUnmarshalJSON(t *testing.T) {
	var sd Duration

	err := json.Unmarshal([]byte(`"1d30m"`), &sd)
	require.NoError(t, err)
	require.Equal(t, Duration(24*time.Hour+30*time.Minute), sd)
}

func TestStringDurationUnmarshalJSONErrors(t *testing.T) {
	var sd Duration

	err := json.Unmarshal([]byte(`"invalid"`), &sd)
	require.Error(t, err)

	sd = 123
	err = json.Unmarshal([]byte(`""`), &sd)
	require.NoError(t, err)
	require.Equal(t, Duration(0), sd)

	sd = 123
	err = json.Unmarshal([]byte(`null`), &sd)
	require.NoError(t, err)
	require.Equal(t, Duration(0), sd)
}
