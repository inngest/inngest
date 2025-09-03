package apiv2

// GenericServiceOptions is a zero-dependency implementation of ServiceOptions
// that is compatible with router.Opts. You can provide your own implementations for
// any services you need, and return nil for services you don't use.
type GenericServiceOptions struct {
	database  any
	encryptor any
}

// ServiceConfig holds all the configuration for creating a portable service
type ServiceConfig struct {
	Database  any
	Encryptor any
}

// NewServiceOptions creates a ServiceOptions implementation.
// You only need to provide implementations for the services you actually use.
// Pass nil for any services you don't need.
func NewServiceOptions(config ServiceConfig) ServiceOptions {
	return &GenericServiceOptions{
		database:  config.Database,
		encryptor: config.Encryptor,
	}
}

// Implement all ServiceOptions methods
func (p *GenericServiceOptions) GetDatabase() any  { return p.database }
func (p *GenericServiceOptions) GetEncryptor() any { return p.encryptor }
