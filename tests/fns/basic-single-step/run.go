package basic_test

import (
	"context"
	"time"

	"github.com/inngest/inngest/tests/testdsl"
)

const cert = `-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIUGBjE2MNb+VxQlaHJMLlVTdoRNZUwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA4MjMxNDAyNDRaFw0yMzA4
MjMxNDAyNDRaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQDhFNszwKnHngsXYb11MRFvIdPT8kHZ6r8tIszydxO3
TdhgwDwW8PREsaB2WucWa9V8GSE/MZUKmeZ6u5E1yxegnk4LZyL9tHxYMmV37MvA
Ai4nfBPag+swyglLgIgHWOZu0KFPoBRNhMNEyEd6yCURICYZQORlbZBHz2ko5Zf0
E9rPfObzMm/DTFWv0QUe9C96KLQvS0pEIHU+/nPSia9m+/aFkXtURk0RBea2mO7e
D6tsofv/BUiVCejRWyFX1MkjxvBCshoXXLyxJFHiUeupaUZwP4hWWW3HEXS8cRfQ
6yu/JJ5qEcVeFJ5hZ7VZLsIuW9Iei1iE60oMZJQH5JkmuWGqNERLeOeWJ4wehSk0
xvXYiFfSv0VOouchLN85FD5KdY6rDc+FFbTjyjZAwRwKNjHABe0X7OFUUNUF/FbD
jvzgVYEAFvg290onOkITbImOSa2RIkdGATJV4K9+JHtoJA8iuEH6J4LJA8Zpe9F3
KV9+fILvU91xDYe0mzbbfH8yKe7JOqLA0vw9G5LjmE9Af0tvkVGhEsI+5buf7UWk
bH7vyV0mOj02+aCygfH3XVsQoDqMBrrqY2vqb5jmj9L8MdyzztdBZcbJis6bsi8D
acjZQGxnPOv0axAnXsUZUTvK0OV0rb+UCbKNFMGNigEedXOk7/ltrOxUPCWlrc7j
RwIDAQABo1MwUTAdBgNVHQ4EFgQUdz8A+9PHUpS2qdiCUNBI5lS3+wIwHwYDVR0j
BBgwFoAUdz8A+9PHUpS2qdiCUNBI5lS3+wIwDwYDVR0TAQH/BAUwAwEB/zANBgkq
hkiG9w0BAQsFAAOCAgEAuypUoWFe8mlhema4bWmNu1LGElmpvs249fjNnzmXZT4+
UKkmHpKfZ2XW2vKVM8PfkrMZvfvu3N8UIkiMYeSn7f2pBN6tTDxzv2suTrm0JfZZ
5kOgiz/aQgG4xQ5DFRLTJrKNph8ixurS343wjm0xvyCboq+HcKPn0dIYQgVnOYq7
N3xqGc9kOCMvBmCvleJZQWZwwHtKpakmlNgyqOI8t5w80mePlxBLJOQlEtlIopzy
O9P5dgOFWPj+K0b12fTHaibLtrCMpRzBWFn5joVD7yBws9aBZPLEKqTf/BigpcuS
8UZ3Yh3DlrLauEQ2nmSlZQM6VYPneO1wLAoAnekU3AAp3Sw0GGnx76PcEzIrUww8
wIltgqNofgjHvGST2TVoVlUvckIUuCP8BQlUSKFAOhSMbGlx5wFyd0rrd16FXRDe
HtbnPI3PnE02nXx5711VtkaEellDx3LbBpycca3xLDlI9FOpcp7mlXEqNjPflYe/
pfCOhgpTdX9OgGWiTf5zZTcETWgAcs32nm4J8LTn2rRV5cdUwvmyPoczN8Q7kfYp
NYpqm+6+vsXK4uol0sLfoKF0CHKWJ3KKNJpXBBFx8l1IRmd6fVgN7BbcqzgjX8hl
eo0uMBNZDfGkgAnnb//Jq5k8C/kri2zTPsgs9T7K1abzsyBkr7RopNQN8kEBP+8=
-----END CERTIFICATE-----`

func init() {
	testdsl.Register(Do)
}

func Do(ctx context.Context) testdsl.Chain {
	return testdsl.Chain{
		testdsl.SendTrigger,
		testdsl.RequireReceiveTrigger,

		// Ensure API publishes event.
		testdsl.RequireLogFields(map[string]any{
			"caller":     "api",
			"event_name": "basic/single-step",
			"message":    "publishing event",
		}),
		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "basic/single-step",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"message": "initializing fn",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "executor",
			"step":    "basic-step-1",
			"message": "executing step",
		}, time.Second*2),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller": "output",
			"output": map[string]any{
				"body": map[string]any{
					"event": "basic/single-step",
					// assert that .env was read accurately.
					"FOO":   "bar please",
					"QUOTE": "quoted key",
					"CERT":  cert,
				},
				"status": 200,
			},
			"message": "step output",
		}, time.Second*2),
		testdsl.RequireNoOutput(`"error"`),
	}
}
