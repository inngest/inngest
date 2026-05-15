package consts

import (
	"time"

	"github.com/google/uuid"
)

const (
	DevServerTempDir = ".inngest"
	DevServerDbFile  = "dev_db.db"
	DevServerRdbFile = "dev_state.rdb"
	// StartDefaultPersistenceInterval is the default interval at which the
	// queue will be snapshotted and persisted to disk.
	StartDefaultPersistenceInterval = time.Second * 60
	// StartMaxQueueChunkSize is the default maximum size of a queue chunk.
	// This is set to be comfortably within the 1GB limit of SQLite.
	StartMaxQueueChunkSize = 1024 * 1024 * 800 // 800MB
	// StartMaxQueueSnapshots is the maximum number of snapshots we keep.
	StartMaxQueueSnapshots  = 5
	DefaultInngestConfigDir = ".inngest"
	SQLiteDbFileName        = "main.db"
	// DevServerHistoryFile is the file where the history is stored.
	//
	// @deprecated Used in the in-memory writer when persiting, though this
	// should not be actively used any more.
	DevServerHistoryFile = "dev_history.json"
)

var (
	// DevServerAccountID is the fixed account ID used internally in the dev server.
	DevServerAccountID = uuid.MustParse("00000000-0000-4000-a000-000000000000")
	DevServerEnvID     = uuid.MustParse("00000000-0000-4000-b000-000000000000")

	DevServerConnectJwtSecret  = []byte("this-does-not-need-to-be-secret")
	DevServerRealtimeJWTSecret = []byte("dev-mode-is-not-secret")
	DevServerRunJWTSecret      = []byte("dev-mode-is-not-secret")
)
