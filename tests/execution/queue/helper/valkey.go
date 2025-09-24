package helper

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewValkeyClient initializes a new valkey client (using redis protocol)
func NewValkeyClient(addr, username, password string, cluster bool) (rueidis.Client, error) {
	return rueidis.NewClient(rueidis.ClientOption{
		InitAddress:       []string{addr},
		Username:          username,
		Password:          password,
		SelectDB:          0,
		DisableCache:      true,
		ForceSingleClient: !cluster, // Force single client mode only when NOT in cluster mode
		Dialer: net.Dialer{
			Timeout: 30 * time.Second,
		},
		ConnWriteTimeout: 30 * time.Second,
	})
}

// ValkeyConfiguration represents the configuration settings for a Valkey server.
// This struct represents all the configuration options from the valkey.conf file.
type ValkeyConfiguration struct {
	// Network settings
	Bind                   string `valkey:"bind,omitempty"`
	BindSourceAddr         string `valkey:"bind-source-addr,omitempty"`
	ProtectedMode          string `valkey:"protected-mode,omitempty"`
	EnableProtectedConfigs string `valkey:"enable-protected-configs,omitempty"`
	EnableDebugCommand     string `valkey:"enable-debug-command,omitempty"`
	EnableModuleCommand    string `valkey:"enable-module-command,omitempty"`
	Port                   int    `valkey:"port,omitempty"`
	TCPBacklog             int    `valkey:"tcp-backlog,omitempty"`
	UnixSocket             string `valkey:"unixsocket,omitempty"`
	UnixSocketGroup        string `valkey:"unixsocketgroup,omitempty"`
	UnixSocketPerm         int    `valkey:"unixsocketperm,omitempty"`
	Timeout                int    `valkey:"timeout,omitempty"`
	TCPKeepalive           int    `valkey:"tcp-keepalive,omitempty"`
	SocketMarkID           int    `valkey:"socket-mark-id,omitempty"`

	// TLS/SSL settings
	TLSPort                int    `valkey:"tls-port,omitempty"`
	TLSCertFile            string `valkey:"tls-cert-file,omitempty"`
	TLSKeyFile             string `valkey:"tls-key-file,omitempty"`
	TLSKeyFilePass         string `valkey:"tls-key-file-pass,omitempty"`
	TLSClientCertFile      string `valkey:"tls-client-cert-file,omitempty"`
	TLSClientKeyFile       string `valkey:"tls-client-key-file,omitempty"`
	TLSClientKeyFilePass   string `valkey:"tls-client-key-file-pass,omitempty"`
	TLSDHParamsFile        string `valkey:"tls-dh-params-file,omitempty"`
	TLSCACertFile          string `valkey:"tls-ca-cert-file,omitempty"`
	TLSCACertDir           string `valkey:"tls-ca-cert-dir,omitempty"`
	TLSAuthClients         string `valkey:"tls-auth-clients,omitempty"`
	TLSReplication         string `valkey:"tls-replication,omitempty"`
	TLSCluster             string `valkey:"tls-cluster,omitempty"`
	TLSProtocols           string `valkey:"tls-protocols,omitempty"`
	TLSCiphers             string `valkey:"tls-ciphers,omitempty"`
	TLSCiphersuites        string `valkey:"tls-ciphersuites,omitempty"`
	TLSPreferServerCiphers string `valkey:"tls-prefer-server-ciphers,omitempty"`
	TLSSessionCaching      string `valkey:"tls-session-caching,omitempty"`
	TLSSessionCacheSize    int    `valkey:"tls-session-cache-size,omitempty"`
	TLSSessionCacheTimeout int    `valkey:"tls-session-cache-timeout,omitempty"`

	// General settings
	Daemonize                  string `valkey:"daemonize,omitempty"`
	Supervised                 string `valkey:"supervised,omitempty"`
	Pidfile                    string `valkey:"pidfile,omitempty"`
	Loglevel                   string `valkey:"loglevel,omitempty"`
	Logfile                    string `valkey:"logfile,omitempty"`
	SyslogEnabled              string `valkey:"syslog-enabled,omitempty"`
	SyslogIdent                string `valkey:"syslog-ident,omitempty"`
	SyslogFacility             string `valkey:"syslog-facility,omitempty"`
	CrashLogEnabled            string `valkey:"crash-log-enabled,omitempty"`
	CrashMemcheckEnabled       string `valkey:"crash-memcheck-enabled,omitempty"`
	Databases                  int    `valkey:"databases,omitempty"`
	AlwaysShowLogo             string `valkey:"always-show-logo,omitempty"`
	HideUserDataFromLog        string `valkey:"hide-user-data-from-log,omitempty"`
	SetProcTitle               string `valkey:"set-proc-title,omitempty"`
	ProcTitleTemplate          string `valkey:"proc-title-template,omitempty"`
	LocaleCollate              string `valkey:"locale-collate,omitempty"`
	ExtendedRedisCompatibility string `valkey:"extended-redis-compatibility,omitempty"`

	// Snapshotting
	Save                    []string `valkey:"save,omitempty"`
	StopWritesOnBgsaveError string   `valkey:"stop-writes-on-bgsave-error,omitempty"`
	RDBCompression          string   `valkey:"rdbcompression,omitempty"`
	RDBChecksum             string   `valkey:"rdbchecksum,omitempty"`
	SanitizeDumpPayload     string   `valkey:"sanitize-dump-payload,omitempty"`
	DBFilename              string   `valkey:"dbfilename,omitempty"`
	RDBDelSyncFiles         string   `valkey:"rdb-del-sync-files,omitempty"`
	Dir                     string   `valkey:"dir,omitempty"`

	// Replication
	ReplicaOf                     string `valkey:"replicaof,omitempty"`
	PrimaryAuth                   string `valkey:"primaryauth,omitempty"`
	PrimaryUser                   string `valkey:"primaryuser,omitempty"`
	ReplicaServeStaleData         string `valkey:"replica-serve-stale-data,omitempty"`
	ReplicaReadOnly               string `valkey:"replica-read-only,omitempty"`
	ReplDisklessSync              string `valkey:"repl-diskless-sync,omitempty"`
	ReplDisklessSyncDelay         int    `valkey:"repl-diskless-sync-delay,omitempty"`
	ReplDisklessSyncMaxReplicas   int    `valkey:"repl-diskless-sync-max-replicas,omitempty"`
	ReplDisklessLoad              string `valkey:"repl-diskless-load,omitempty"`
	DualChannelReplicationEnabled string `valkey:"dual-channel-replication-enabled,omitempty"`
	ReplPingReplicaPeriod         int    `valkey:"repl-ping-replica-period,omitempty"`
	ReplTimeout                   int    `valkey:"repl-timeout,omitempty"`
	ReplDisableTCPNodelay         string `valkey:"repl-disable-tcp-nodelay,omitempty"`
	ReplBacklogSize               string `valkey:"repl-backlog-size,omitempty"`
	ReplBacklogTTL                int    `valkey:"repl-backlog-ttl,omitempty"`
	ReplicaPriority               int    `valkey:"replica-priority,omitempty"`
	PropagationErrorBehavior      string `valkey:"propagation-error-behavior,omitempty"`
	ReplicaIgnoreDiskWriteErrors  string `valkey:"replica-ignore-disk-write-errors,omitempty"`
	ReplicaAnnounced              string `valkey:"replica-announced,omitempty"`
	MinReplicasToWrite            int    `valkey:"min-replicas-to-write,omitempty"`
	MinReplicasMaxLag             int    `valkey:"min-replicas-max-lag,omitempty"`
	ReplicaAnnounceIP             string `valkey:"replica-announce-ip,omitempty"`
	ReplicaAnnouncePort           int    `valkey:"replica-announce-port,omitempty"`

	// Keys tracking
	TrackingTableMaxKeys int `valkey:"tracking-table-max-keys,omitempty"`

	// Security
	RequirePass      string `valkey:"requirepass,omitempty"`
	ACLFile          string `valkey:"aclfile,omitempty"`
	ACLPubsubDefault string `valkey:"acl-pubsub-default,omitempty"`
	ACLLogMaxLen     int    `valkey:"acllog-max-len,omitempty"`

	// Clients
	MaxClients int `valkey:"maxclients,omitempty"`

	// Memory management
	MaxMemory                 string `valkey:"maxmemory,omitempty"`
	MaxMemoryPolicy           string `valkey:"maxmemory-policy,omitempty"`
	MaxMemorySamples          int    `valkey:"maxmemory-samples,omitempty"`
	MaxMemoryEvictionTenacity int    `valkey:"maxmemory-eviction-tenacity,omitempty"`
	ReplicaIgnoreMaxMemory    string `valkey:"replica-ignore-maxmemory,omitempty"`
	ActiveExpireEffort        int    `valkey:"active-expire-effort,omitempty"`

	// Lazy freeing
	LazyfreeLazyEviction  string `valkey:"lazyfree-lazy-eviction,omitempty"`
	LazyfreeLazyExpire    string `valkey:"lazyfree-lazy-expire,omitempty"`
	LazyfreeLazyServerDel string `valkey:"lazyfree-lazy-server-del,omitempty"`
	ReplicaLazyFlush      string `valkey:"replica-lazy-flush,omitempty"`
	LazyfreeLazyUserDel   string `valkey:"lazyfree-lazy-user-del,omitempty"`
	LazyfreeLazyUserFlush string `valkey:"lazyfree-lazy-user-flush,omitempty"`

	// Threaded I/O
	IOThreads            int `valkey:"io-threads,omitempty"`
	PrefetchBatchMaxSize int `valkey:"prefetch-batch-max-size,omitempty"`

	// Kernel OOM control
	OOMScoreAdj       string `valkey:"oom-score-adj,omitempty"`
	OOMScoreAdjValues []int  `valkey:"oom-score-adj-values,omitempty"`

	// Kernel transparent hugepage
	DisableTHP string `valkey:"disable-thp,omitempty"`

	// Append only mode
	AppendOnly               string `valkey:"appendonly,omitempty"`
	AppendFilename           string `valkey:"appendfilename,omitempty"`
	AppendDirname            string `valkey:"appenddirname,omitempty"`
	AppendFsync              string `valkey:"appendfsync,omitempty"`
	NoAppendfsyncOnRewrite   string `valkey:"no-appendfsync-on-rewrite,omitempty"`
	AutoAOFRewritePercentage int    `valkey:"auto-aof-rewrite-percentage,omitempty"`
	AutoAOFRewriteMinSize    string `valkey:"auto-aof-rewrite-min-size,omitempty"`
	AOFLoadTruncated         string `valkey:"aof-load-truncated,omitempty"`
	AOFUseRDBPreamble        string `valkey:"aof-use-rdb-preamble,omitempty"`
	AOFTimestampEnabled      string `valkey:"aof-timestamp-enabled,omitempty"`

	// Shutdown
	ShutdownTimeout   int    `valkey:"shutdown-timeout,omitempty"`
	ShutdownOnSigint  string `valkey:"shutdown-on-sigint,omitempty"`
	ShutdownOnSigterm string `valkey:"shutdown-on-sigterm,omitempty"`

	// Non-deterministic long blocking commands
	BusyReplyThreshold int `valkey:"busy-reply-threshold,omitempty"`

	// Valkey cluster
	ClusterEnabled                  string `valkey:"cluster-enabled,omitempty"`
	ClusterConfigFile               string `valkey:"cluster-config-file,omitempty"`
	ClusterNodeTimeout              int    `valkey:"cluster-node-timeout,omitempty"`
	ClusterPort                     int    `valkey:"cluster-port,omitempty"`
	ClusterReplicaValidityFactor    int    `valkey:"cluster-replica-validity-factor,omitempty"`
	ClusterMigrationBarrier         int    `valkey:"cluster-migration-barrier,omitempty"`
	ClusterAllowReplicaMigration    string `valkey:"cluster-allow-replica-migration,omitempty"`
	ClusterRequireFullCoverage      string `valkey:"cluster-require-full-coverage,omitempty"`
	ClusterReplicaNoFailover        string `valkey:"cluster-replica-no-failover,omitempty"`
	ClusterAllowReadsWhenDown       string `valkey:"cluster-allow-reads-when-down,omitempty"`
	ClusterAllowPubsubshardWhenDown string `valkey:"cluster-allow-pubsubshard-when-down,omitempty"`
	ClusterLinkSendbufLimit         int    `valkey:"cluster-link-sendbuf-limit,omitempty"`
	ClusterAnnounceHostname         string `valkey:"cluster-announce-hostname,omitempty"`
	ClusterAnnounceHumanNodename    string `valkey:"cluster-announce-human-nodename,omitempty"`
	ClusterPreferredEndpointType    string `valkey:"cluster-preferred-endpoint-type,omitempty"`
	ClusterBlacklistTTL             int    `valkey:"cluster-blacklist-ttl,omitempty"`
	ClusterSlotStatsEnabled         string `valkey:"cluster-slot-stats-enabled,omitempty"`
	ClusterAnnounceIP               string `valkey:"cluster-announce-ip,omitempty"`
	ClusterAnnounceClientIPv4       string `valkey:"cluster-announce-client-ipv4,omitempty"`
	ClusterAnnounceClientIPv6       string `valkey:"cluster-announce-client-ipv6,omitempty"`
	ClusterAnnouncePort             int    `valkey:"cluster-announce-port,omitempty"`
	ClusterAnnounceTLSPort          int    `valkey:"cluster-announce-tls-port,omitempty"`
	ClusterAnnounceBusPort          int    `valkey:"cluster-announce-bus-port,omitempty"`

	// Slow log
	SlowlogLogSlowerThan int `valkey:"slowlog-log-slower-than,omitempty"`
	SlowlogMaxLen        int `valkey:"slowlog-max-len,omitempty"`

	// Latency monitor
	LatencyMonitorThreshold int `valkey:"latency-monitor-threshold,omitempty"`

	// Latency tracking
	LatencyTracking                string   `valkey:"latency-tracking,omitempty"`
	LatencyTrackingInfoPercentiles []string `valkey:"latency-tracking-info-percentiles,omitempty"`

	// Event notification
	NotifyKeyspaceEvents string `valkey:"notify-keyspace-events,omitempty"`

	// Advanced config
	HashMaxListpackEntries       int      `valkey:"hash-max-listpack-entries,omitempty"`
	HashMaxListpackValue         int      `valkey:"hash-max-listpack-value,omitempty"`
	ListMaxListpackSize          int      `valkey:"list-max-listpack-size,omitempty"`
	ListCompressDepth            int      `valkey:"list-compress-depth,omitempty"`
	SetMaxIntsetEntries          int      `valkey:"set-max-intset-entries,omitempty"`
	SetMaxListpackEntries        int      `valkey:"set-max-listpack-entries,omitempty"`
	SetMaxListpackValue          int      `valkey:"set-max-listpack-value,omitempty"`
	ZSetMaxListpackEntries       int      `valkey:"zset-max-listpack-entries,omitempty"`
	ZSetMaxListpackValue         int      `valkey:"zset-max-listpack-value,omitempty"`
	HLLSparseMaxBytes            int      `valkey:"hll-sparse-max-bytes,omitempty"`
	StreamNodeMaxBytes           int      `valkey:"stream-node-max-bytes,omitempty"`
	StreamNodeMaxEntries         int      `valkey:"stream-node-max-entries,omitempty"`
	ActiveRehashing              string   `valkey:"activerehashing,omitempty"`
	ClientOutputBufferLimit      []string `valkey:"client-output-buffer-limit,omitempty"`
	ClientQueryBufferLimit       string   `valkey:"client-query-buffer-limit,omitempty"`
	MaxMemoryClients             string   `valkey:"maxmemory-clients,omitempty"`
	ProtoMaxBulkLen              string   `valkey:"proto-max-bulk-len,omitempty"`
	Hz                           int      `valkey:"hz,omitempty"`
	DynamicHz                    string   `valkey:"dynamic-hz,omitempty"`
	AOFRewriteIncrementalFsync   string   `valkey:"aof-rewrite-incremental-fsync,omitempty"`
	RDBSaveIncrementalFsync      string   `valkey:"rdb-save-incremental-fsync,omitempty"`
	LFULogFactor                 int      `valkey:"lfu-log-factor,omitempty"`
	LFUDecayTime                 int      `valkey:"lfu-decay-time,omitempty"`
	MaxNewConnectionsPerCycle    int      `valkey:"max-new-connections-per-cycle,omitempty"`
	MaxNewTLSConnectionsPerCycle int      `valkey:"max-new-tls-connections-per-cycle,omitempty"`

	// Active defragmentation
	ActiveDefrag               string `valkey:"activedefrag,omitempty"`
	ActiveDefragIgnoreBytes    string `valkey:"active-defrag-ignore-bytes,omitempty"`
	ActiveDefragThresholdLower int    `valkey:"active-defrag-threshold-lower,omitempty"`
	ActiveDefragThresholdUpper int    `valkey:"active-defrag-threshold-upper,omitempty"`
	ActiveDefragCycleMin       int    `valkey:"active-defrag-cycle-min,omitempty"`
	ActiveDefragCycleMax       int    `valkey:"active-defrag-cycle-max,omitempty"`
	ActiveDefragMaxScanFields  int    `valkey:"active-defrag-max-scan-fields,omitempty"`
	JemallocBGThread           string `valkey:"jemalloc-bg-thread,omitempty"`

	// CPU affinity
	ServerCPUList     string `valkey:"server-cpulist,omitempty"`
	BioCPUList        string `valkey:"bio-cpulist,omitempty"`
	AOFRewriteCPUList string `valkey:"aof-rewrite-cpulist,omitempty"`
	BgsaveCPUList     string `valkey:"bgsave-cpulist,omitempty"`

	// Ignore warnings
	IgnoreWarnings string `valkey:"ignore-warnings,omitempty"`

	// Availability zone
	AvailabilityZone string `valkey:"availability-zone,omitempty"`
}

type ValkeyContainer struct {
	testcontainers.Container

	Addr     string
	Username string
	Password string
}

// ValkeyOption represents a configuration option for the Valkey container
type ValkeyOption func(*valkeyConfig)

// valkeyConfig holds the configuration for starting a Valkey container
type valkeyConfig struct {
	customConfig *ValkeyConfiguration

	// maxMemory in bytes
	maxMemory int64

	// Custom Docker image (defaults to valkey/valkey:8.0.1)
	image string

	// cluster mode (defaults to false for standalone mode)
	cluster bool
}

// WithValkeyConfiguration sets a custom Valkey configuration
func WithValkeyConfiguration(config *ValkeyConfiguration) ValkeyOption {
	return func(vc *valkeyConfig) {
		vc.customConfig = config
	}
}

// WithValkeyMaxMemory sets the maximum memory for Valkey
func WithValkeyMaxMemory(maxMemory int64) ValkeyOption {
	return func(vc *valkeyConfig) {
		vc.maxMemory = maxMemory
	}
}

// WithValkeyImage sets a custom Docker image for Valkey
func WithValkeyImage(image string) ValkeyOption {
	return func(vc *valkeyConfig) {
		vc.image = image
	}
}

// WithValkeyCluster enables or disables cluster mode for Valkey
func WithValkeyCluster(enabled bool) ValkeyOption {
	return func(vc *valkeyConfig) {
		vc.cluster = enabled
	}
}

// formatValkeyConfig converts ValkeyConfiguration to valkey.conf format
func formatValkeyConfig(config *ValkeyConfiguration) string {
	var configStr string

	// For simplicity, we'll handle a few key config options
	// In a production implementation, you'd use reflection to handle all fields
	if config.Port != 0 {
		configStr += fmt.Sprintf("port %d\n", config.Port)
	}
	if config.Bind != "" {
		configStr += fmt.Sprintf("bind %s\n", config.Bind)
	}
	if config.RequirePass != "" {
		configStr += fmt.Sprintf("requirepass %s\n", config.RequirePass)
	}
	if config.MaxMemory != "" {
		configStr += fmt.Sprintf("maxmemory %s\n", config.MaxMemory)
	}
	if config.AppendOnly != "" {
		configStr += fmt.Sprintf("appendonly %s\n", config.AppendOnly)
	}
	if config.ClusterEnabled != "" {
		configStr += fmt.Sprintf("cluster-enabled %s\n", config.ClusterEnabled)
	}
	if config.ClusterConfigFile != "" {
		configStr += fmt.Sprintf("cluster-config-file %s\n", config.ClusterConfigFile)
	}
	if config.ClusterNodeTimeout != 0 {
		configStr += fmt.Sprintf("cluster-node-timeout %d\n", config.ClusterNodeTimeout)
	}

	// Add any other important configuration options...

	return configStr
}

func StartValkey(t *testing.T, opts ...ValkeyOption) (*ValkeyContainer, error) {
	// Apply options
	config := &valkeyConfig{
		image: "valkey/valkey:8.0.1", // Default image
	}
	for _, opt := range opts {
		opt(config)
	}
	ctx := t.Context()

	passwd := "hello"
	port := 6379

	// Build container request based on configuration
	req := testcontainers.ContainerRequest{
		Image:         config.image,
		ImagePlatform: "linux/amd64",
		ExposedPorts:  []string{fmt.Sprintf("%d/tcp", port)},
		WaitingFor:    wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = config.maxMemory
		},
	}

	// Handle custom configuration if provided
	var tempDir string
	if config.customConfig != nil {
		// Create temporary directory for config
		var err error
		tempDir, err = os.MkdirTemp("", "valkey-config-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		// Clean up temp dir when function exits
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// Format and write configuration
		configContent := formatValkeyConfig(config.customConfig)

		// Add default settings if not already in custom config
		if config.customConfig.RequirePass == "" {
			configContent += fmt.Sprintf("requirepass %s\n", passwd)
		}
		if config.customConfig.Port == 0 {
			configContent += fmt.Sprintf("port %d\n", port)
		}
		if config.cluster {
			configContent += "cluster-enabled yes\n"
		}
		configContent += "appendonly yes\n"

		// Write config to temporary file
		configPath := filepath.Join(tempDir, "valkey.conf")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write config file: %w", err)
		}

		// Configure container with bind mount
		req.HostConfigModifier = func(hc *container.HostConfig) {
			hc.Mounts = []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: configPath,
					Target: "/usr/local/etc/valkey/valkey.conf",
				},
			}
			hc.Memory = config.maxMemory
		}
		req.Cmd = []string{"valkey-server", "/etc/valkey/valkey.conf"}
	} else {
		// Use default command line configuration
		cmd := []string{
			"valkey-server",
			"--port", fmt.Sprintf("%d", port),
			"--requirepass", passwd,
			"--appendonly", "yes",
		}
		if config.cluster {
			cmd = append(cmd, "--cluster-enabled", "yes")
		}
		req.Cmd = cmd
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get the mapped port for external access
	mappedPort, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Get the host addr
	host, err := container.Host(ctx)
	require.NoError(t, err)

	connectAddr := fmt.Sprintf("%s:%s", host, mappedPort.Port())
	fmt.Printf("Connecting to: %s\n", connectAddr)

	// Wait a bit for the container to finish starting up after the log appears
	<-time.After(time.Second)

	for range 5 {
		// Create client with the mapped port
		rc, err := NewValkeyClient(connectAddr, "", passwd, config.cluster)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			<-time.After(time.Second)
			continue
		}

		pong, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
		if err != nil {
			fmt.Printf("PING ERROR: %v\n", err)
			rc.Close()
			<-time.After(time.Second)
			continue
		}

		fmt.Println("RESP:", pong)
		if pong == "PONG" {
			// Initialize cluster slots only if cluster mode is enabled
			if config.cluster {
				_, err = rc.Do(ctx, rc.B().Arbitrary("CLUSTER", "ADDSLOTSRANGE", "0", "16383").Build()).ToString()
				if err != nil {
					if err.Error() != "" && (err.Error() == "ERR Slot 0 is already busy" || err.Error() == "ERR Slot 0 is already assigned in the cluster" || err.Error() == "ERR already assigned") {
						fmt.Println("Slots already assigned")
					} else {
						rc.Close()
						return nil, fmt.Errorf("failed to initialize cluster slots: %w", err)
					}
				}
			}

			rc.Close()
			return &ValkeyContainer{
				Container: container,
				Addr:      connectAddr,
				Password:  passwd,
			}, nil
		}
		rc.Close()

		<-time.After(time.Second)
	}

	return nil, fmt.Errorf("valkey is not available")
}

