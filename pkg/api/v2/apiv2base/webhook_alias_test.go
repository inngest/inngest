package apiv2base

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestWebhookAliasRequestSchema(t *testing.T) {
	for _, path := range []string{"/api/v2/webhooks", "/api/v2/env/webhooks"} {
		t.Run(path, func(t *testing.T) {
			message := getProtoMessageForPath(path, http.MethodPost)
			require.NotNil(t, message)
			require.Equal(t, protoreflect.FullName("api.v2.CreateWebhookRequest"), message.ProtoReflect().Descriptor().FullName())
		})
	}
}
