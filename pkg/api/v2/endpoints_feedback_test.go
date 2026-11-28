package apiv2

import (
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSubmitFeedbackCloudOnly(t *testing.T) {
	s := NewService(ServiceOptions{})
	response, err := s.SubmitFeedback(t.Context(), &apiv2.SubmitFeedbackRequest{Feedback: "hello"})
	require.Nil(t, response)
	require.ErrorContains(t, err, `inngest api --prod submit-feedback --feedback`)
}
