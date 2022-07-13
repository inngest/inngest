package models

type Environment string

const (
	EnvironmentProduction Environment = "prod"
	EnvironmentTest       Environment = "test"
)

func (e Environment) String() string {
	switch e {
	case EnvironmentProduction:
		return EnvironmentProduction.String()
	case EnvironmentTest:
		return EnvironmentTest.String()
	default:
		return ""
	}
}
