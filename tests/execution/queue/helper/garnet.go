package helper

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

//
// Helper
//

// NewRedisClient initializes a new redis client
func NewRedisClient(addr, username, password string) (rueidis.Client, error) {
	return rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{addr},
		Username:     username,
		Password:     password,
		SelectDB:     0,
		DisableCache: true, // garnet doesn't support client side caching
		Dialer: net.Dialer{
			Timeout: 30 * time.Second, // Increased connection timeout
		},
		ConnWriteTimeout: 30 * time.Second, // Increased write timeout for large keys
	})
}

// GarnetConfig represents the configuration settings for a Garnet server.
// Fields with zero values will be omitted from the JSON output.
type GarnetConfiguration struct {
	// Port to run server on
	Port int `json:"Port,omitempty"`

	// Whitespace or comma separated string of IP addresses to bind server to (default: any)
	Address string `json:"Address,omitempty"`

	// Port that this node advertises to other nodes to connect to for gossiping.
	ClusterAnnouncePort int `json:"ClusterAnnouncePort,omitempty"`

	// IP address that this node advertises to other nodes to connect to for gossiping.
	ClusterAnnounceIp string `json:"ClusterAnnounceIp,omitempty"`

	// Total log memory used in bytes (rounds down to power of 2)
	MemorySize string `json:"MemorySize,omitempty"`

	// Size of each page in bytes (rounds down to power of 2)
	PageSize string `json:"PageSize,omitempty"`

	// Size of each log segment in bytes on disk (rounds down to power of 2)
	SegmentSize string `json:"SegmentSize,omitempty"`

	// Start size of hash index in bytes (rounds down to power of 2)
	IndexSize string `json:"IndexSize,omitempty"`

	// Max size of hash index in bytes (rounds down to power of 2)
	IndexMaxSize string `json:"IndexMaxSize,omitempty"`

	// Percentage of log memory that is kept mutable
	MutablePercent int `json:"MutablePercent,omitempty"`

	// Enable read cache for faster access to on-disk records
	EnableReadCache bool `json:"EnableReadCache,omitempty"`

	// Total read cache log memory used in bytes (rounds down to power of 2)
	ReadCacheMemorySize string `json:"ReadCacheMemorySize,omitempty"`

	// Size of each read cache page in bytes (rounds down to power of 2)
	ReadCachePageSize string `json:"ReadCachePageSize,omitempty"`

	// Object store heap memory size in bytes (Sum of size taken up by all object instances in the heap)
	ObjectStoreHeapMemorySize string `json:"ObjectStoreHeapMemorySize,omitempty"`

	// Object store log memory used in bytes
	ObjectStoreLogMemorySize string `json:"ObjectStoreLogMemorySize,omitempty"`

	// Size of each object store page in bytes (rounds down to power of 2)
	ObjectStorePageSize string `json:"ObjectStorePageSize,omitempty"`

	// Size of each object store log segment in bytes on disk (rounds down to power of 2)
	ObjectStoreSegmentSize string `json:"ObjectStoreSegmentSize,omitempty"`

	// Start size of object store hash index in bytes (rounds down to power of 2)
	ObjectStoreIndexSize string `json:"ObjectStoreIndexSize,omitempty"`

	// Max size of object store hash index in bytes (rounds down to power of 2)
	ObjectStoreIndexMaxSize string `json:"ObjectStoreIndexMaxSize,omitempty"`

	// Percentage of object store log memory that is kept mutable
	ObjectStoreMutablePercent int `json:"ObjectStoreMutablePercent,omitempty"`

	// Enables object store read cache for faster access to on-disk records
	EnableObjectStoreReadCache bool `json:"EnableObjectStoreReadCache,omitempty"`

	// Total object store read cache log memory used in bytes (rounds down to power of 2)
	ObjectStoreReadCacheLogMemorySize string `json:"ObjectStoreReadCacheLogMemorySize,omitempty"`

	// Size of each object store read cache page in bytes (rounds down to power of 2)
	ObjectStoreReadCachePageSize string `json:"ObjectStoreReadCachePageSize,omitempty"`

	// Object store read cache heap memory size in bytes
	ObjectStoreReadCacheHeapMemorySize string `json:"ObjectStoreReadCacheHeapMemorySize,omitempty"`

	// Enable tiering of records (hybrid log) to storage
	EnableStorageTier bool `json:"EnableStorageTier,omitempty"`

	// When records are read from the main store's in-memory immutable region or storage device, copy them to the tail of the log.
	CopyReadsToTail bool `json:"CopyReadsToTail,omitempty"`

	// When records are read from the object store's in-memory immutable region or storage device, copy them to the tail of the log.
	ObjectStoreCopyReadsToTail bool `json:"ObjectStoreCopyReadsToTail,omitempty"`

	// Storage directory for tiered records (hybrid log)
	LogDir string `json:"LogDir,omitempty"`

	// Storage directory for checkpoints. Uses logdir if unspecified.
	CheckpointDir string `json:"CheckpointDir,omitempty"`

	// Recover from latest checkpoint and log, if present.
	Recover bool `json:"Recover,omitempty"`

	// Disable pub/sub feature on server.
	DisablePubSub bool `json:"DisablePubSub,omitempty"`

	// Enable incremental snapshots.
	EnableIncrementalSnapshots bool `json:"EnableIncrementalSnapshots,omitempty"`

	// Page size of log used for pub/sub (rounds down to power of 2)
	PubSubPageSize string `json:"PubSubPageSize,omitempty"`

	// Disable support for data structure objects.
	DisableObjects bool `json:"DisableObjects,omitempty"`

	// Enable cluster.
	EnableCluster bool `json:"EnableCluster,omitempty"`

	// Start with clean cluster config.
	CleanClusterConfig bool `json:"CleanClusterConfig,omitempty"`

	// Number of parallel migrate tasks to spawn when SLOTS or SLOTSRANGE option is used.
	ParallelMigrateTaskCount int `json:"ParallelMigrateTaskCount,omitempty"`

	// Fast migration options.
	FastMigrate bool `json:"FastMigrate,omitempty"`

	// Authentication mode of Garnet.
	AuthenticationMode string `json:"AuthenticationMode,omitempty"`

	// Authentication string for password authentication.
	Password string `json:"Password,omitempty"`

	// Username to authenticate intra-cluster communication with.
	ClusterUsername string `json:"ClusterUsername,omitempty"`

	// Password to authenticate intra-cluster communication with.
	ClusterPassword string `json:"ClusterPassword,omitempty"`

	// External ACL user file.
	AclFile string `json:"AclFile,omitempty"`

	// The authority of AAD authentication.
	AadAuthority string `json:"AadAuthority,omitempty"`

	// The audiences of AAD token for AAD authentication.
	AadAudiences string `json:"AadAudiences,omitempty"`

	// The issuers of AAD token for AAD authentication.
	AadIssuers string `json:"AadIssuers,omitempty"`

	// The authorized client app Ids for AAD authentication.
	AuthorizedAadApplicationIds string `json:"AuthorizedAadApplicationIds,omitempty"`

	// Whether to validate username as ObjectId or a valid Group objectId if present in claims.
	AadValidateUsername bool `json:"AadValidateUsername,omitempty"`

	// Enable write ahead logging (append-only file).
	EnableAOF bool `json:"EnableAOF,omitempty"`

	// Total AOF memory buffer used in bytes.
	AofMemorySize string `json:"AofMemorySize,omitempty"`

	// Size of each AOF page in bytes.
	AofPageSize string `json:"AofPageSize,omitempty"`

	// AOF replication (safe tail address) refresh frequency in milliseconds.
	AofReplicationRefreshFrequencyMs int `json:"AofReplicationRefreshFrequencyMs,omitempty"`

	// Subscriber (safe tail address) refresh frequency in milliseconds (for pub-sub).
	SubscriberRefreshFrequencyMs int `json:"SubscriberRefreshFrequencyMs,omitempty"`

	// Write ahead logging (append-only file) commit issue frequency in milliseconds.
	CommitFrequencyMs int `json:"CommitFrequencyMs,omitempty"`

	// Wait for AOF to flush the commit before returning results to client.
	WaitForCommit bool `json:"WaitForCommit,omitempty"`

	// Maximum size of AOF after which unsafe truncation will be applied.
	AofSizeLimit string `json:"AofSizeLimit,omitempty"`

	// Background hybrid log compaction frequency in seconds.
	CompactionFrequencySecs int `json:"CompactionFrequencySecs,omitempty"`

	// Frequency in seconds for the background task to perform object collection.
	ExpiredObjectCollectionFrequencySecs int `json:"ExpiredObjectCollectionFrequencySecs,omitempty"`

	// Hybrid log compaction type.
	CompactionType string `json:"CompactionType,omitempty"`

	// Forcefully delete the inactive segments immediately after the compaction strategy is applied.
	CompactionForceDelete bool `json:"CompactionForceDelete,omitempty"`

	// Number of log segments created on disk before compaction triggers.
	CompactionMaxSegments int `json:"CompactionMaxSegments,omitempty"`

	// Number of object store log segments created on disk before compaction triggers.
	ObjectStoreCompactionMaxSegments int `json:"ObjectStoreCompactionMaxSegments,omitempty"`

	// Enable Lua scripts on server.
	EnableLua bool `json:"EnableLua,omitempty"`

	// Run Lua scripts as a transaction.
	LuaTransactionMode bool `json:"LuaTransactionMode,omitempty"`

	// Percent of cluster nodes to gossip with at each gossip iteration.
	GossipSamplePercent int `json:"GossipSamplePercent,omitempty"`

	// Cluster mode gossip protocol per node sleep (in seconds) delay.
	GossipDelay int `json:"GossipDelay,omitempty"`

	// Cluster node timeout in seconds.
	ClusterTimeout int `json:"ClusterTimeout,omitempty"`

	// How frequently to flush cluster config unto disk to persist updates.
	ClusterConfigFlushFrequencyMs int `json:"ClusterConfigFlushFrequencyMs,omitempty"`

	// Name for the client target host when using TLS connections in cluster mode.
	ClusterTlsClientTargetHost string `json:"ClusterTlsClientTargetHost,omitempty"`

	// Enable TLS.
	EnableTLS bool `json:"EnableTLS,omitempty"`

	// TLS certificate file name.
	CertFileName string `json:"CertFileName,omitempty"`

	// TLS certificate password.
	CertPassword string `json:"CertPassword,omitempty"`

	// TLS certificate subject name.
	CertSubjectName string `json:"CertSubjectName,omitempty"`

	// TLS certificate refresh frequency in seconds.
	CertificateRefreshFrequency int `json:"CertificateRefreshFrequency,omitempty"`

	// Whether client TLS certificate is required by the server.
	ClientCertificateRequired bool `json:"ClientCertificateRequired,omitempty"`

	// Whether server TLS certificate is required by clients established on the server side.
	ServerCertificateRequired bool `json:"ServerCertificateRequired,omitempty"`

	// Certificate revocation check mode for certificate validation.
	CertificateRevocationCheckMode string `json:"CertificateRevocationCheckMode,omitempty"`

	// Full path of file with issuer certificate for validation.
	IssuerCertificatePath string `json:"IssuerCertificatePath,omitempty"`

	// Track latency of various events.
	LatencyMonitor bool `json:"LatencyMonitor,omitempty"`

	// Threshold (microseconds) for logging command in the slow log.
	SlowLogThreshold int `json:"SlowLogThreshold,omitempty"`

	// Maximum number of slow log entries to keep.
	SlowLogMaxEntries int `json:"SlowLogMaxEntries,omitempty"`

	// Metrics sampling frequency in seconds.
	MetricsSamplingFrequency int `json:"MetricsSamplingFrequency,omitempty"`

	// Enabling quiet mode does not print server version and text art.
	QuietMode bool `json:"QuietMode,omitempty"`

	// Logging level.
	LogLevel string `json:"LogLevel,omitempty"`

	// Frequency (in seconds) of logging.
	LoggingFrequency string `json:"LoggingFrequency,omitempty"`

	// Disable console logger.
	DisableConsoleLogger bool `json:"DisableConsoleLogger,omitempty"`

	// Enable file logger and write to the specified path.
	FileLogger string `json:"FileLogger,omitempty"`

	// Minimum worker threads in thread pool.
	ThreadPoolMinThreads int `json:"ThreadPoolMinThreads,omitempty"`

	// Maximum worker threads in thread pool.
	ThreadPoolMaxThreads int `json:"ThreadPoolMaxThreads,omitempty"`

	// Minimum IO completion threads in thread pool.
	ThreadPoolMinIOCompletionThreads int `json:"ThreadPoolMinIOCompletionThreads,omitempty"`

	// Maximum IO completion threads in thread pool.
	ThreadPoolMaxIOCompletionThreads int `json:"ThreadPoolMaxIOCompletionThreads,omitempty"`

	// Maximum number of simultaneously active network connections.
	NetworkConnectionLimit int `json:"NetworkConnectionLimit,omitempty"`

	// Use Azure Page Blobs for storage instead of local storage.
	UseAzureStorage bool `json:"UseAzureStorage,omitempty"`

	// The connection string to use when establishing connection to Azure Blobs Storage.
	AzureStorageConnectionString string `json:"AzureStorageConnectionString,omitempty"`

	// The URI to use when establishing connection to Azure Blobs Storage.
	AzureStorageServiceUri string `json:"AzureStorageServiceUri,omitempty"`

	// The managed identity to use when establishing connection to Azure Blobs Storage.
	AzureStorageManagedIdentity string `json:"AzureStorageManagedIdentity,omitempty"`

	// Whether and by how much should we throttle the disk IO for checkpoints.
	CheckpointThrottleFlushDelayMs int `json:"CheckpointThrottleFlushDelayMs,omitempty"`

	// Use FastCommit when writing AOF.
	EnableFastCommit bool `json:"EnableFastCommit,omitempty"`

	// Throttle FastCommit to write metadata once every K commits.
	FastCommitThrottleFreq int `json:"FastCommitThrottleFreq,omitempty"`

	// Throttle the maximum outstanding network sends per session.
	NetworkSendThrottleMax int `json:"NetworkSendThrottleMax,omitempty"`

	// Whether we use scatter gather IO for MGET or a batch of contiguous GET operations.
	EnableScatterGatherGet bool `json:"EnableScatterGatherGet,omitempty"`

	// Whether and by how much (milliseconds) should we throttle the replica sync.
	ReplicaSyncDelayMs int `json:"ReplicaSyncDelayMs,omitempty"`

	// Throttle ClusterAppendLog when replica.AOFTailAddress - ReplicationOffset > ReplicationOffsetMaxLag.
	ReplicationOffsetMaxLag int `json:"ReplicationOffsetMaxLag,omitempty"`

	// Use main-memory replication model.
	MainMemoryReplication bool `json:"MainMemoryReplication,omitempty"`

	// Use fast-aof-truncate replication model.
	FastAofTruncate bool `json:"FastAofTruncate,omitempty"`

	// Used with main-memory replication model to take on demand checkpoint.
	OnDemandCheckpoint bool `json:"OnDemandCheckpoint,omitempty"`

	// Whether diskless replication is enabled or not.
	ReplicaDisklessSync bool `json:"ReplicaDisklessSync,omitempty"`

	// Delay in diskless replication sync in seconds.
	ReplicaDisklessSyncDelay int `json:"ReplicaDisklessSyncDelay,omitempty"`

	// Timeout in seconds for replication sync operations.
	ReplicaSyncTimeout int `json:"ReplicaSyncTimeout,omitempty"`

	// Timeout in seconds for replication attach operations.
	ReplicaAttachTimeout int `json:"ReplicaAttachTimeout,omitempty"`

	// AOF replay size threshold for diskless replication.
	ReplicaDisklessSyncFullSyncAofThreshold string `json:"ReplicaDisklessSyncFullSyncAofThreshold,omitempty"`

	// With main-memory replication, use null device for AOF.
	UseAofNullDevice bool `json:"UseAofNullDevice,omitempty"`

	// Use native device on Linux for local storage.
	UseNativeDeviceLinux bool `json:"UseNativeDeviceLinux,omitempty"`

	// The sizes of records in each revivification bin.
	RevivBinRecordSizes string `json:"RevivBinRecordSizes,omitempty"`

	// The number of records in each revivification bin.
	RevivBinRecordCounts string `json:"RevivBinRecordCounts,omitempty"`

	// Fraction of mutable in-memory log space eligible for revivification.
	RevivifiableFraction float64 `json:"RevivifiableFraction,omitempty"`

	// A shortcut to specify revivification with default power-of-2-sized bins.
	EnableRevivification bool `json:"EnableRevivification,omitempty"`

	// Number of next-higher bins to search if the search cannot be satisfied in the best-fitting bin.
	RevivNumberOfBinsToSearch int `json:"RevivNumberOfBinsToSearch,omitempty"`

	// Number of records to scan for best fit after finding first fit.
	RevivBinBestFitScanLimit int `json:"RevivBinBestFitScanLimit,omitempty"`

	// Revivify tombstoned records in tag chains only.
	RevivInChainOnly bool `json:"RevivInChainOnly,omitempty"`

	// Number of records in the single free record bin for the object store.
	RevivObjBinRecordCount int `json:"RevivObjBinRecordCount,omitempty"`

	// Limit of items to return in one iteration of *SCAN command.
	ObjectScanCountLimit int `json:"ObjectScanCountLimit,omitempty"`

	// Enable DEBUG command for clients - no/local/yes.
	EnableDebugCommand string `json:"EnableDebugCommand,omitempty"`

	// Enable MODULE command for clients - no/local/yes.
	EnableModuleCommand string `json:"EnableModuleCommand,omitempty"`

	// Protected mode.
	ProtectedMode string `json:"ProtectedMode,omitempty"`

	// List of directories on server from which custom command binaries can be loaded.
	ExtensionBinPaths string `json:"ExtensionBinPaths,omitempty"`

	// Allow loading custom commands from digitally unsigned assemblies.
	ExtensionAllowUnsignedAssemblies bool `json:"ExtensionAllowUnsignedAssemblies,omitempty"`

	// Index resize check frequency in seconds.
	IndexResizeFrequencySecs int `json:"IndexResizeFrequencySecs,omitempty"`

	// Overflow bucket count over total index size in percentage to trigger index resize.
	IndexResizeThreshold int `json:"IndexResizeThreshold,omitempty"`

	// List of module paths to be loaded at startup.
	LoadModuleCS string `json:"LoadModuleCS,omitempty"`

	// Fails if encounters error during AOF replay or checkpointing.
	FailOnRecoveryError bool `json:"FailOnRecoveryError,omitempty"`

	// Skips crc64 validation in restore command.
	SkipRDBRestoreChecksumValidation bool `json:"SkipRDBRestoreChecksumValidation,omitempty"`

	// Lua memory management mode.
	LuaMemoryManagementMode string `json:"LuaMemoryManagementMode,omitempty"`

	// Lua memory limits.
	LuaScriptMemoryLimit string `json:"LuaScriptMemoryLimit,omitempty"`

	// Timeout on Lua scripts in milliseconds.
	LuaScriptTimeoutMs int `json:"LuaScriptTimeoutMs,omitempty"`

	// Allow redis.log(...) to write to the Garnet logs.
	LuaLoggingMode string `json:"LuaLoggingMode,omitempty"`

	// Allow all built in and redis.* functions by default.
	LuaAllowedFunctions string `json:"LuaAllowedFunctions,omitempty"`

	// Unix socket address path to bind the server to.
	UnixSocketPath string `json:"UnixSocketPath,omitempty"`

	// Unix socket permissions in octal (Unix platforms only).
	UnixSocketPermission int `json:"UnixSocketPermission,omitempty"`

	// Max number of logical databases allowed in a single Garnet server instance.
	MaxDatabases int `json:"MaxDatabases,omitempty"`

	// Frequency of background scan for expired key deletion, in seconds.
	ExpiredKeyDeletionScanFrequencySecs int `json:"ExpiredKeyDeletionScanFrequencySecs,omitempty"`

	// Maximum frequency cluster Replicas will attempt to re-establish replicating their Primaries, in seconds.
	ClusterReplicationReestablishmentTimeout int `json:"ClusterReplicationReestablishmentTimeout,omitempty"`

	// If a Cluster Replica has on disk data, if that data should be loaded on restart.
	ClusterReplicaResumeWithData bool `json:"ClusterReplicaResumeWithData,omitempty"`
}

type GarnetContainer struct {
	testcontainers.Container

	Addr     string
	Username string
	Password string
}

// GarnetOption represents a configuration option for the Garnet container
type GarnetOption func(*garnetConfig)

// garnetConfig holds the configuration for starting a Garnet container
type garnetConfig struct {
	customConfig *GarnetConfiguration

	// maxMemory in bytes
	maxMemory int64
}

// WithConfiguration sets a custom Garnet configuration
func WithConfiguration(config *GarnetConfiguration) GarnetOption {
	return func(gc *garnetConfig) {
		gc.customConfig = config
	}
}

func WithMaxMemory(maxMemory int64) GarnetOption {
	return func(gc *garnetConfig) {
		gc.maxMemory = maxMemory
	}
}

func StartGarnet(t *testing.T, opts ...GarnetOption) (*GarnetContainer, error) {
	// Apply options
	config := &garnetConfig{}
	for _, opt := range opts {
		opt(config)
	}
	ctx := t.Context()

	passwd := "hello"
	port := 6379

	// Build container request based on configuration
	req := testcontainers.ContainerRequest{
		Image:         "ghcr.io/microsoft/garnet:1.0.83",
		ImagePlatform: "linux/amd64", // Necessary for Lua support
		ExposedPorts:  []string{fmt.Sprintf("%d/tcp", port)},
		WaitingFor:    wait.ForLog("* Ready to accept connections").WithStartupTimeout(30 * time.Second),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = config.maxMemory
		},
	}

	// Handle custom configuration if provided
	var tempDir string
	if config.customConfig != nil {
		// Create temporary directory for config
		var err error
		tempDir, err = os.MkdirTemp("", "garnet-config-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		// Clean up temp dir when function exits
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// Marshal the configuration to JSON
		configJSON, err := json.MarshalIndent(config.customConfig, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}

		// Write config to temporary file
		configPath := filepath.Join(tempDir, "garnet.conf")
		err = os.WriteFile(configPath, configJSON, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write config file: %w", err)
		}

		// Configure container with bind mount
		req.HostConfigModifier = func(hc *container.HostConfig) {
			hc.Mounts = []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: configPath,
					Target: "/etc/garnet/garnet.conf",
				},
			}
		}
		req.ConfigModifier = func(c *container.Config) {
			c.Cmd = strslice.StrSlice{
				"--config-import-path", "/etc/garnet/garnet.conf",
				fmt.Sprintf("--port=%d", port),
				"--cluster",
				"--auth", "Password",
				fmt.Sprintf("--password=%s", passwd),
				"--aof",
			}
		}
	} else {
		// Use default command line configuration
		req.ConfigModifier = func(c *container.Config) {
			// https://microsoft.github.io/garnet/docs/getting-started/configuration#garnetconf
			c.Cmd = strslice.StrSlice{
				fmt.Sprintf("--port=%d", port),
				"--cluster",
				"--auth", "Password",
				fmt.Sprintf("--password=%s", passwd),
				"--aof",
			}
		}
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
		rc, err := NewRedisClient(connectAddr, "", passwd)
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
			// Initialize cluster slots
			_, err = rc.Do(ctx, rc.B().Arbitrary("CLUSTER", "ADDSLOTSRANGE", "0", "16383").Build()).ToString()
			if err != nil {
				if err.Error() != "" && (err.Error() == "ERR Slot 0 is already busy" || err.Error() == "ERR already assigned") {
					fmt.Println("Slots already assigned")
				} else {
					rc.Close()
					return nil, fmt.Errorf("failed to initialize cluster slots: %w", err)
				}
			}

			rc.Close()
			return &GarnetContainer{
				Container: container,
				Addr:      connectAddr,
				Password:  passwd,
			}, nil
		}
		rc.Close()

		<-time.After(time.Second)
	}

	return nil, fmt.Errorf("garnet is not available")
}
