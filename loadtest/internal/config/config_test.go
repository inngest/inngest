package config

import (
	"strings"
	"testing"
)

func TestValidateDefaultsModeToDev(t *testing.T) {
	c := Defaults()
	c.Target.Mode = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.Target.Mode != ModeDev {
		t.Errorf("expected Mode to default to %q, got %q", ModeDev, c.Target.Mode)
	}
}

func TestValidateSelfHostedRequiresSigningKey(t *testing.T) {
	c := Defaults()
	c.Target.Mode = ModeSelfHosted

	// nil
	c.Target.SigningKey = nil
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "signingKey") {
		t.Errorf("expected signingKey error, got %v", err)
	}

	// empty string
	empty := ""
	c.Target.SigningKey = &empty
	err = c.Validate()
	if err == nil || !strings.Contains(err.Error(), "signingKey") {
		t.Errorf("expected signingKey error, got %v", err)
	}

	// valid
	k := "signkey-prod-xxx"
	c.Target.SigningKey = &k
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
}

func TestValidateUnknownModeRejected(t *testing.T) {
	c := Defaults()
	c.Target.Mode = "cloud"
	if err := c.Validate(); err == nil {
		t.Errorf("expected error for unknown mode")
	}
}

func TestValidateDevModeNoSigningKeyOK(t *testing.T) {
	c := Defaults()
	c.Target.Mode = ModeDev
	c.Target.SigningKey = nil
	if err := c.Validate(); err != nil {
		t.Errorf("dev mode should not require signing key: %v", err)
	}
}
