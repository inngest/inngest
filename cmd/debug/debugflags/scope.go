package debugflags

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func AccountEnvFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "account-id",
			Usage: "Account UUID that owns the function",
		},
		&cli.StringFlag{
			Name:  "env-id",
			Usage: "Environment UUID that owns the function",
		},
	}
}

func FunctionFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "function-id",
		Usage: "Function UUID that owns the debounce",
	}
}

func AccountEnv(cmd *cli.Command) (string, string, error) {
	accountID, err := RequiredUUID(cmd, "account-id")
	if err != nil {
		return "", "", err
	}

	envID, err := RequiredUUID(cmd, "env-id")
	if err != nil {
		return "", "", err
	}

	return accountID, envID, nil
}

func RequiredUUID(cmd *cli.Command, name string) (string, error) {
	val := cmd.String(name)
	if val == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if _, err := uuid.Parse(val); err != nil {
		return "", fmt.Errorf("invalid %s: %w", name, err)
	}
	return val, nil
}
