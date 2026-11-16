package computecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

const codeVerificationRequired = "verification_required"

// accessWait is how long deploy waits for verification.
var accessWait = 30 * time.Minute

func (c *client) computeAccess(ctx context.Context) (*apiv2.ComputeAccess, error) {
	resp := &apiv2.ComputeAccessResponse{}
	if err := c.do(ctx, http.MethodGet, "/compute/access", nil, resp); err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}

// requireComputeAccess stops a deploy before it builds when the account can't
// deploy.  when the account has to be verified first, it opens the verify
// page and waits in a terminal; elsewhere it prints the link and stops.
func requireComputeAccess(ctx context.Context, stdin io.Reader, stdout io.Writer, api *client) error {
	access, err := api.computeAccess(ctx)
	var apiErr *apiError
	if errors.As(err, &apiErr) && (apiErr.statusCode == http.StatusNotImplemented ||
		apiErr.statusCode == http.StatusNotFound && !strings.Contains(apiErr.msg, "env_not_found")) {
		// the server doesn't have the check yet.  the deploy itself still
		// refuses an account that can't deploy.
		return nil
	}
	if err != nil {
		return err
	}
	if access.GetAllowed() {
		return nil
	}
	if access.GetCode() != codeVerificationRequired || !isTerminal(stdin, stdout) {
		return accessError(access)
	}
	_ = cliauth.OpenBrowser(access.GetVerifyUrl())
	return waitForVerification(ctx, stdout, api, access.GetVerifyUrl())
}

// waitForVerification polls until the account can deploy.
func waitForVerification(ctx context.Context, stdout io.Writer, api *client, verifyURL string) error {
	fmt.Fprintf(stdout, "This account must be verified before it can deploy.  An account admin can verify it at:\n\n  %s\n\nWaiting for verification...\n", verifyURL)
	// a deadline on ctx would also cut off the poll in flight, which then
	// fails with a bare context error.
	deadline := time.After(accessWait)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			return fmt.Errorf("the account wasn't verified within %s; run inngest deploy again once it is", accessWait)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		access, err := api.computeAccess(ctx)
		if err != nil {
			return err
		}
		if access.GetAllowed() {
			fmt.Fprintf(stdout, "Verified.\n\n")
			return nil
		}
		if access.GetCode() != codeVerificationRequired {
			return accessError(access)
		}
	}
}

func accessError(access *apiv2.ComputeAccess) error {
	return fmt.Errorf("%s: %s", access.GetCode(), access.GetMessage())
}
