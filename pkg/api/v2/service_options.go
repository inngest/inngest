package apiv2

// GenericServiceOptions is a zero-dependency implementation of ServiceOptions
// that is compatible with router.Opts. You can provide your own implementations for
// any services you need, and return nil for services you don't use.
type GenericServiceOptions struct {
	actionConfig               any
	authn                      any
	bigQuery                   any
	cancellationReadWriter     any
	clerk                      any
	conditionalConnectTracer   any
	connectGatewayRetriever    any
	connectHistoryRPC          any
	connectRequestAuther       any
	connectRequestStateManager any
	connectTokenSigner         any
	database                   any
	dbCache                    any
	encryptor                  any
	entitlementProvider        any
	entitlements               any
	eventReader                any
	executor                   any
	fdb                        any
	fnReader                   any
	historyReader              any
	log                        any
	metrics                    any
	metricsRPC                 any
	publisher                  any
	queueShards                any
	readiness                  any
	readOnlyDB                 any
	redis                      any
	replayStore                any
	unshardedClient            any
}

// ServiceConfig holds all the configuration for creating a portable service
type ServiceConfig struct {
	ActionConfig               any
	Authn                      any
	BigQuery                   any
	CancellationReadWriter     any
	Clerk                      any
	ConditionalConnectTracer   any
	ConnectGatewayRetriever    any
	ConnectHistoryRPC          any
	ConnectRequestAuther       any
	ConnectRequestStateManager any
	ConnectTokenSigner         any
	Database                   any
	DBCache                    any
	Encryptor                  any
	EntitlementProvider        any
	Entitlements               any
	EventReader                any
	Executor                   any
	FDB                        any
	FnReader                   any
	HistoryReader              any
	Log                        any
	Metrics                    any
	MetricsRPC                 any
	Publisher                  any
	QueueShards                any
	Readiness                  any
	ReadOnlyDB                 any
	Redis                      any
	ReplayStore                any
	UnshardedClient            any
}

// NewServiceOptions creates a ServiceOptions implementation.
// You only need to provide implementations for the services you actually use.
// Pass nil for any services you don't need.
func NewServiceOptions(config ServiceConfig) ServiceOptions {
	return &GenericServiceOptions{
		actionConfig:               config.ActionConfig,
		authn:                      config.Authn,
		bigQuery:                   config.BigQuery,
		cancellationReadWriter:     config.CancellationReadWriter,
		clerk:                      config.Clerk,
		conditionalConnectTracer:   config.ConditionalConnectTracer,
		connectGatewayRetriever:    config.ConnectGatewayRetriever,
		connectHistoryRPC:          config.ConnectHistoryRPC,
		connectRequestAuther:       config.ConnectRequestAuther,
		connectRequestStateManager: config.ConnectRequestStateManager,
		connectTokenSigner:         config.ConnectTokenSigner,
		database:                   config.Database,
		dbCache:                    config.DBCache,
		encryptor:                  config.Encryptor,
		entitlementProvider:        config.EntitlementProvider,
		entitlements:               config.Entitlements,
		eventReader:                config.EventReader,
		executor:                   config.Executor,
		fdb:                        config.FDB,
		fnReader:                   config.FnReader,
		historyReader:              config.HistoryReader,
		log:                        config.Log,
		metrics:                    config.Metrics,
		metricsRPC:                 config.MetricsRPC,
		publisher:                  config.Publisher,
		queueShards:                config.QueueShards,
		readiness:                  config.Readiness,
		readOnlyDB:                 config.ReadOnlyDB,
		redis:                      config.Redis,
		replayStore:                config.ReplayStore,
		unshardedClient:            config.UnshardedClient,
	}
}

// Implement all ServiceOptions methods
func (p *GenericServiceOptions) GetActionConfig() any             { return p.actionConfig }
func (p *GenericServiceOptions) GetAuthn() any                    { return p.authn }
func (p *GenericServiceOptions) GetBigQuery() any                 { return p.bigQuery }
func (p *GenericServiceOptions) GetCancellationReadWriter() any   { return p.cancellationReadWriter }
func (p *GenericServiceOptions) GetClerk() any                    { return p.clerk }
func (p *GenericServiceOptions) GetConditionalConnectTracer() any { return p.conditionalConnectTracer }
func (p *GenericServiceOptions) GetConnectGatewayRetriever() any  { return p.connectGatewayRetriever }
func (p *GenericServiceOptions) GetConnectHistoryRPC() any        { return p.connectHistoryRPC }
func (p *GenericServiceOptions) GetConnectRequestAuther() any     { return p.connectRequestAuther }
func (p *GenericServiceOptions) GetConnectRequestStateManager() any {
	return p.connectRequestStateManager
}
func (p *GenericServiceOptions) GetConnectTokenSigner() any  { return p.connectTokenSigner }
func (p *GenericServiceOptions) GetDB() any                  { return p.database }
func (p *GenericServiceOptions) GetDBCache() any             { return p.dbCache }
func (p *GenericServiceOptions) GetEncryptor() any           { return p.encryptor }
func (p *GenericServiceOptions) GetEntitlementProvider() any { return p.entitlementProvider }
func (p *GenericServiceOptions) GetEntitlements() any        { return p.entitlements }
func (p *GenericServiceOptions) GetEventReader() any         { return p.eventReader }
func (p *GenericServiceOptions) GetExecutor() any            { return p.executor }
func (p *GenericServiceOptions) GetFDB() any                 { return p.fdb }
func (p *GenericServiceOptions) GetFnReader() any            { return p.fnReader }
func (p *GenericServiceOptions) GetHistoryReader() any       { return p.historyReader }
func (p *GenericServiceOptions) GetLog() any                 { return p.log }
func (p *GenericServiceOptions) GetMetrics() any             { return p.metrics }
func (p *GenericServiceOptions) GetMetricsRPC() any          { return p.metricsRPC }
func (p *GenericServiceOptions) GetPublisher() any           { return p.publisher }
func (p *GenericServiceOptions) GetQueueShards() any         { return p.queueShards }
func (p *GenericServiceOptions) GetReadiness() any           { return p.readiness }
func (p *GenericServiceOptions) GetReadOnlyDB() any          { return p.readOnlyDB }
func (p *GenericServiceOptions) GetRedis() any               { return p.redis }
func (p *GenericServiceOptions) GetReplayStore() any         { return p.replayStore }
func (p *GenericServiceOptions) GetUnshardedClient() any     { return p.unshardedClient }
