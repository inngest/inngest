package apiv2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestWriteSandboxFileDataUsesNumericBytesWritten(t *testing.T) {
	encoded, err := protojson.Marshal(&WriteSandboxFileData{
		Path:         "/tmp/message.txt",
		BytesWritten: 5,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"path":"/tmp/message.txt","bytesWritten":5}`, string(encoded))
}
