package state

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/fs"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)
)

var (
	ErrWorkerGroupNotFound = fmt.Errorf("worker group not found")
	ErrGatewayNotFound     = fmt.Errorf("gateway not found")
)

func init() {
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
}

func createScript(script string) (*rueidis.Lua, error) {
	// Add any includes.
	items := include.FindAllStringSubmatch(script, -1)
	if len(items) > 0 {
		// Replace each include
		for _, include := range items {
			byt, err := embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
			if err != nil {
				return nil, fmt.Errorf("error reading redis lua include: %w", err)
			}
			script = strings.ReplaceAll(script, include[0], string(byt))
		}
	}

	return rueidis.NewLuaScript(script), nil
}

func readRedisScripts(path string, entries []fs.DirEntry) {
	for _, e := range entries {
		// NOTE: When using embed go always uses forward slashes as a path
		// prefix. filepath.Join uses OS-specific prefixes which fails on
		// windows, so we construct the path using Sprintf for all platforms
		if e.IsDir() {
			entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			readRedisScripts(path+"/"+e.Name(), entries)
			continue
		}

		byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading redis lua script: %w", err))
		}

		val := string(byt)
		script, err := createScript(val)
		if err != nil {
			panic(err)
		}

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")
		scripts[name] = script
	}
}

type redisConnectionStateManager struct {
	c      clockwork.Clock
	client rueidis.Client
	logger *slog.Logger
}

type RedisStateManagerOpt struct {
	Clock clockwork.Clock
}

func NewRedisConnectionStateManager(client rueidis.Client, opts ...RedisStateManagerOpt) *redisConnectionStateManager {
	c := clockwork.NewRealClock()
	if len(opts) > 0 && opts[0].Clock != nil {
		c = opts[0].Clock
	}

	return &redisConnectionStateManager{
		client: client,
		logger: logger.StdlibLogger(context.Background()),
		c:      c,
	}
}

func (r *redisConnectionStateManager) GetConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) (*connpb.ConnMetadata, error) {
	key := r.connectionHash(envID)
	cmd := r.client.B().Hget().Key(key).Field(connId.String()).Build()

	res, err := r.client.Do(ctx, cmd).ToString()
	if err != nil {
		if errors.Is(err, rueidis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	conn := connpb.ConnMetadata{}
	if err := json.Unmarshal([]byte(res), &conn); err != nil {
		return nil, err
	}

	return &conn, nil
}

func (r *redisConnectionStateManager) GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	key := r.connectionHash(envID)
	cmd := r.client.B().Hvals().Key(key).Build()

	res, err := r.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	conns := []*connpb.ConnMetadata{}
	for _, meta := range res {
		var conn connpb.ConnMetadata
		if err := json.Unmarshal([]byte(meta), &conn); err != nil {
			return nil, err
		}
		conns = append(conns, &conn)
	}

	return conns, nil
}

func (r *redisConnectionStateManager) GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	key := r.connIndexByApp(envId, appID)

	connIds, err := r.client.Do(ctx, r.client.B().Smembers().Key(key).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	if len(connIds) == 0 {
		return nil, nil
	}

	res, err := r.client.Do(ctx, r.client.B().Hmget().Key(r.connectionHash(envId)).Field(connIds...).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	conns := []*connpb.ConnMetadata{}
	for _, meta := range res {
		var conn connpb.ConnMetadata
		if err := json.Unmarshal([]byte(meta), &conn); err != nil {
			return nil, err
		}
		conns = append(conns, &conn)
	}

	return conns, nil
}

func (r *redisConnectionStateManager) GetConnectionsByGroupID(ctx context.Context, envID uuid.UUID, groupID string) ([]*connpb.ConnMetadata, error) {
	keys := []string{
		r.connectionHash(envID),
		r.connIndexByGroup(envID, groupID),
	}
	args := []string{}

	res, err := scripts["get_conns_by_group"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error retrieving conns by group: %w", err)
	}

	if len(res) == 0 {
		return nil, nil
	}

	conns := []*connpb.ConnMetadata{}
	for _, cs := range res {
		var conn connpb.ConnMetadata
		if err := json.Unmarshal([]byte(cs), &conn); err != nil {
			return nil, fmt.Errorf("error deserializing conn metadata: %w", err)
		}
		conns = append(conns, &conn)
	}

	return conns, nil
}

func (r *redisConnectionStateManager) sortGroups(groups []*WorkerGroup) {
	slices.SortStableFunc(groups, func(a, b *WorkerGroup) int {
		// If a is synced but b isn't, a should come first
		if a.AppID != nil && b.AppID == nil {
			return -1
		}

		// If b is synced but a isn't, b should come first
		if a.AppID == nil && b.AppID != nil {
			return 1
		}

		return strings.Compare(a.Hash, b.Hash)
	})
}

func (r *redisConnectionStateManager) UpsertConnection(ctx context.Context, conn *Connection, status connpb.ConnectionStatus, lastHeartbeatAt time.Time) error {
	// Reduce variations by sorting groups based on syncs
	sortedGroups := make([]*WorkerGroup, 0, len(conn.Groups))
	for _, group := range conn.Groups {
		sortedGroups = append(sortedGroups, group)
	}
	r.sortGroups(sortedGroups)

	// Map App ID -> Worker Group hash (this is only available after syncing)
	syncedWorkerGroups := make(map[string]string)
	for _, group := range sortedGroups {
		if group.AppID != nil {
			syncedWorkerGroups[group.AppID.String()] = group.Hash
		}
	}

	// Map app name -> Worker group hash (this is set even when the group is not synced)
	allWorkerGroups := make(map[string]string)
	for _, group := range sortedGroups {
		allWorkerGroups[group.AppName] = group.Hash
	}

	meta := &connpb.ConnMetadata{
		Id:                 conn.ConnectionId.String(),
		SyncedWorkerGroups: syncedWorkerGroups,
		AllWorkerGroups:    allWorkerGroups,
		InstanceId:         conn.Data.InstanceId,
		Status:             status,
		SdkLanguage:        conn.Data.SdkLanguage,
		SdkVersion:         conn.Data.SdkVersion,
		Attributes:         conn.Data.SystemAttributes,
		GatewayId:          conn.GatewayId.String(),
		LastHeartbeatAt:    timestamppb.New(lastHeartbeatAt),
	}

	// NOTE: redis_state.StrSlice format the data in a non JSON way, not sure why
	var serializedConnection string
	{
		byt, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("error serializing connection metadata: %w", err)
		}
		serializedConnection = string(byt)
	}

	/*
		In this function, we dynamically build a Lua script. We do this because we want atomic
		execution of connection upserts. If we relax this constraint, we could run individual commands in sequence.

		There are rules to follow to avoid performance problems:
		- We must limit the number of unique scripts:
			Redis hashes and caches Lua scripts in memory. This cache is not cleared,
			so adding an infinite number of unique scripts will lead to Redis running
			out of memory. Thus, we must limit variations.

		To this end:
		- We must not include information that vary per request in the script template
		- When building dynamic segments using a range loop, we must limit the max. number if iterations
	*/
	keysDefs := []string{
		"local indexConnectionsByEnvIdKey = KEYS[1]",
		"local indexWorkerGroupsByEnvIdKey = KEYS[2]",
	}
	keys := []string{
		// Upsert conn
		r.connectionHash(conn.EnvID),

		// Upsert worker groups
		r.workerGroupHash(conn.EnvID),
	}

	argDefs := []string{
		"local connID = ARGV[1]",
		"local serializedConn = ARGV[2]",
	}
	args := []string{
		meta.Id,
		serializedConnection,
	}

	groupUpserts := make([]string, 0)
	indexUpdates := make([]string, 0)

	{
		i := 0
		for _, group := range sortedGroups {
			// Push groupId
			groupIdVarName := fmt.Sprintf("groupId%d", i)
			argDefs = append(argDefs, fmt.Sprintf("local %s = ARGV[%d]", groupIdVarName, len(argDefs)+1))
			args = append(args, group.Hash)

			// Push serialized group
			workerGroupVarName := fmt.Sprintf("workerGroup%d", i)
			argDefs = append(argDefs, fmt.Sprintf("local %s = ARGV[%d]", workerGroupVarName, len(argDefs)+1))

			byt, err := json.Marshal(group)
			if err != nil {
				return fmt.Errorf("error serializing worker group data: %w", err)
			}
			args = append(args, string(byt))

			groupUpserts = append(groupUpserts, fmt.Sprintf(`-- Upsert group %d
-- Store the group if it doesn't exist yet
redis.call("HSETNX", indexWorkerGroupsByEnvIdKey, %s, %s)`, i, groupIdVarName, workerGroupVarName))

			// Push index updates
			indexVarName := fmt.Sprintf("indexConnectionsByGroupIdKey%d", i)
			keysDefs = append(keysDefs, fmt.Sprintf("local %s = KEYS[%d]", indexVarName, len(keysDefs)+1))
			keys = append(keys, r.connIndexByGroup(conn.EnvID, group.Hash))
			indexUpdates = append(indexUpdates, fmt.Sprintf(`-- Update index %s
redis.call("SADD", %s, connID)`, indexVarName, indexVarName))

			if group.AppID != nil {
				indexVarName := fmt.Sprintf("indexConnectionsByAppIdKey%d", i)
				keysDefs = append(keysDefs, fmt.Sprintf("local %s = KEYS[%d]", indexVarName, len(keysDefs)+1))
				keys = append(keys, r.connIndexByApp(conn.EnvID, *group.AppID))
				indexUpdates = append(indexUpdates, fmt.Sprintf(`-- Update index %s
redis.call("SADD", %s, connID)`, indexVarName, indexVarName))
			}

			i++
		}
	}

	script, err := createScript(fmt.Sprintf(`
%s

%s

-- $include(ends_with.lua)

-- Store the connection metadata in a map
redis.call("HSET", indexConnectionsByEnvIdKey, connID, serializedConn)

%s

%s

return 0
`,
		strings.Join(keysDefs, "\n"),
		strings.Join(argDefs, "\n"),

		strings.Join(groupUpserts, "\n\n"),
		strings.Join(indexUpdates, "\n\n"),
	))
	if err != nil {
		return fmt.Errorf("could not create upsert script: %w", err)
	}

	resp, err := script.Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return err
	}

	switch resp {
	case 0:
		return nil

	default:
		return fmt.Errorf("unknown status when storing connection metadata: %d", resp)
	}
}

func (r *redisConnectionStateManager) DeleteConnection(ctx context.Context, envID uuid.UUID, connID ulid.ULID) error {
	existingConn, err := r.GetConnection(ctx, envID, connID)
	if err != nil {
		return fmt.Errorf("could not get connection: %w", err)
	}

	if existingConn == nil {
		return nil
	}

	// Fetch all associated worker groups
	groupHashes := make([]string, 0, len(existingConn.AllWorkerGroups))
	for _, groupHash := range existingConn.AllWorkerGroups {
		groupHashes = append(groupHashes, groupHash)
	}

	groups, err := r.GetWorkerGroupsByHash(ctx, envID, groupHashes)
	if err != nil {
		return fmt.Errorf("could not get worker groups for connection: %w", err)
	}

	/*
		In this function, we dynamically build a Lua script. We do this because we want atomic
		execution of connection upserts. If we relax this constraint, we could run individual commands in sequence.

		There are rules to follow to avoid performance problems:
		- We must limit the number of unique scripts:
			Redis hashes and caches Lua scripts in memory. This cache is not cleared,
			so adding an infinite number of unique scripts will lead to Redis running
			out of memory. Thus, we must limit variations.

		To this end:
		- We must not include information that vary per request in the script template
		- When building dynamic segments using a range loop, we must limit the max. number if iterations
	*/
	keysDefs := []string{
		"local indexConnectionsByEnvIdKey = KEYS[1]",
		"local indexWorkerGroupsByEnvIdKey = KEYS[2]",
	}
	keys := []string{
		// Upsert conn
		r.connectionHash(envID),

		// Upsert worker groups
		r.workerGroupHash(envID),
	}

	argDefs := []string{
		"local connID = ARGV[1]",
	}
	args := []string{
		connID.String(),
	}

	indexUpdates := make([]string, 0)

	emptyGroupCleanup := make([]string, 0, len(groups))

	{
		i := 0
		for _, group := range groups {
			// Push groupId
			groupIdVarName := fmt.Sprintf("groupId%d", i)
			argDefs = append(argDefs, fmt.Sprintf("local %s = ARGV[%d]", groupIdVarName, len(argDefs)+1))
			args = append(args, group.Hash)

			// Push index updates
			connectionsByGroupIndexVarName := fmt.Sprintf("indexConnectionsByGroupIdKey%d", i)
			keysDefs = append(keysDefs, fmt.Sprintf("local %s = KEYS[%d]", connectionsByGroupIndexVarName, len(keysDefs)+1))
			keys = append(keys, r.connIndexByGroup(envID, group.Hash))
			indexUpdates = append(indexUpdates, fmt.Sprintf(`-- Remove connection from group index %s
redis.call("SREM", %s, connID)`, connectionsByGroupIndexVarName, connectionsByGroupIndexVarName))

			emptyGroupCleanup = append(emptyGroupCleanup, fmt.Sprintf(`-- If the group is empty, remove it
local scount = tonumber(redis.call("SCARD", %s))
if scount == 0 then
  redis.call("HDEL", indexWorkerGroupsByEnvIdKey, %s)
end`, connectionsByGroupIndexVarName, groupIdVarName))

			if group.AppID != nil {
				indexVarName := fmt.Sprintf("indexConnectionsByAppIdKey%d", i)
				keysDefs = append(keysDefs, fmt.Sprintf("local %s = KEYS[%d]", indexVarName, len(keysDefs)+1))
				keys = append(keys, r.connIndexByApp(envID, *group.AppID))
				indexUpdates = append(indexUpdates, fmt.Sprintf(`-- Remove connection from app index %s
redis.call("SREM", %s, connID)`, indexVarName, indexVarName))
			}

			i++
		}
	}

	script, err := createScript(fmt.Sprintf(`
%s

%s

-- $include(ends_with.lua)

-- Remove the connection from the map
redis.call("HDEL", indexConnectionsByEnvIdKey, connID)

%s

%s

return 0
`,
		strings.Join(keysDefs, "\n"),
		strings.Join(argDefs, "\n"),

		strings.Join(indexUpdates, "\n\n"),

		strings.Join(emptyGroupCleanup, "\n\n"),
	))
	if err != nil {
		return fmt.Errorf("could not create delete script: %w", err)
	}

	status, err := script.Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("could not delete connection: %w", err)
	}

	switch status {
	case 0:
		return nil

	default:
		return fmt.Errorf("unknow status when deleting connection: %w", err)
	}
}

func (r *redisConnectionStateManager) GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error) {
	key := r.workerGroupHash(envID)
	cmd := r.client.B().Hget().Key(key).Field(hash).Build()

	byt, err := r.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrWorkerGroupNotFound
		}
		return nil, fmt.Errorf("error retrieving worker group: %w", err)
	}

	var group WorkerGroup
	if err := json.Unmarshal(byt, &group); err != nil {
		return nil, fmt.Errorf("error deserializing worker group: %w", err)
	}

	return &group, nil
}

func (r *redisConnectionStateManager) GetWorkerGroupsByHash(ctx context.Context, envID uuid.UUID, hashes []string) ([]WorkerGroup, error) {
	res, err := r.client.Do(ctx, r.client.B().Hmget().Key(r.workerGroupHash(envID)).Field(hashes...).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	groups := make([]WorkerGroup, 0)
	for i, meta := range res {
		if meta == "" {
			return nil, fmt.Errorf("could not find group %q: %w", hashes[i], ErrWorkerGroupNotFound)
		}
		var group WorkerGroup
		if err := json.Unmarshal([]byte(meta), &group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (r *redisConnectionStateManager) UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error {
	byt, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("error serializing worker group for update: %w", err)
	}

	key := r.workerGroupHash(envID)
	cmd := r.client.B().Hset().Key(key).FieldValue().FieldValue(group.Hash, string(byt)).Build()

	if err := r.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("error updating worker group: %w", err)
	}

	return nil
}

// connectionHash points to the hash resolving connections by environment.
func (r *redisConnectionStateManager) connectionHash(envID uuid.UUID) string {
	return fmt.Sprintf("{%s}:conns", envID)
}

// connectionHash points to the index hash resolving connections by app ID.
func (r *redisConnectionStateManager) connIndexByApp(envID uuid.UUID, appId uuid.UUID) string {
	return fmt.Sprintf("{%s}:conns_appid:%s", envID.String(), appId.String())
}

// workerGroupHash points to the hash resolving worker groups by environment ID.
func (r *redisConnectionStateManager) workerGroupHash(envID uuid.UUID) string {
	return fmt.Sprintf("{%s}:groups", envID.String())
}

func (r *redisConnectionStateManager) connIndexByGroup(envID uuid.UUID, groupID string) string {
	return fmt.Sprintf("{%s}:groups:%s", envID.String(), groupID)
}

// gatewaysHashKey returns the key for the global gateways hash.
// Gateways are not scoped to any environment, so the Redis hash tag will be global.
// This also means that gateways cannot be accessed in the same script as other environment-scoped keys.
func (r *redisConnectionStateManager) gatewaysHashKey() string {
	return "{connect}:gateways"
}

func (r *redisConnectionStateManager) UpsertGateway(ctx context.Context, gateway *Gateway) error {
	marshaled, err := json.Marshal(gateway)
	if err != nil {
		return fmt.Errorf("could not marshal gateway state: %w", err)
	}

	res := r.client.Do(
		ctx,
		r.client.B().Hset().Key(r.gatewaysHashKey()).FieldValue().FieldValue(gateway.Id.String(), string(marshaled)).Build(),
	)
	if err := res.Error(); err != nil {
		return fmt.Errorf("could not set gateway state: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) DeleteGateway(ctx context.Context, gatewayId ulid.ULID) error {
	res := r.client.Do(
		ctx,
		r.client.B().Hdel().Key(r.gatewaysHashKey()).Field(gatewayId.String()).Build(),
	)
	if err := res.Error(); err != nil {
		return fmt.Errorf("could not delete gateway state: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) GetGateway(ctx context.Context, gatewayId ulid.ULID) (*Gateway, error) {
	gatewayBytes, err := r.client.Do(
		ctx,
		r.client.B().Hget().Key(r.gatewaysHashKey()).Field(gatewayId.String()).Build(),
	).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrGatewayNotFound
		}

		return nil, fmt.Errorf("could not get gateway state: %w", err)
	}

	var gateway Gateway
	if err := json.Unmarshal(gatewayBytes, &gateway); err != nil {
		return nil, fmt.Errorf("could not unmarshal gateway state: %w", err)
	}

	return &gateway, nil
}
