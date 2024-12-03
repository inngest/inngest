package state

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/fs"
	"log/slog"
	"regexp"
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
	ConnDeletedWithGroupErr = fmt.Errorf("group deleted with conn")
	WorkerGroupNotFoundErr  = fmt.Errorf("worker group not found")
	GatewayNotFoundErr      = fmt.Errorf("gateway not found")
)

func init() {
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
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

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")
		val := string(byt)

		// Add any includes.
		items := include.FindAllStringSubmatch(val, -1)
		if len(items) > 0 {
			// Replace each include
			for _, include := range items {
				byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
				if err != nil {
					panic(fmt.Errorf("error reading redis lua include: %w", err))
				}
				val = strings.ReplaceAll(val, include[0], string(byt))
			}
		}

		scripts[name] = rueidis.NewLuaScript(val)
	}
}

type redisConnectionStateManager struct {
	client rueidis.Client
	logger *slog.Logger
}

func NewRedisConnectionStateManager(client rueidis.Client) *redisConnectionStateManager {
	return &redisConnectionStateManager{
		client: client,
		logger: logger.StdlibLogger(context.Background()),
	}
}

func (r redisConnectionStateManager) SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error {
	idempotencyKey := fmt.Sprintf("{%s}:idempotency:%s", appId, requestId)
	res := r.client.Do(
		ctx,
		r.client.B().Set().Key(idempotencyKey).Value("1").Nx().Ex(time.Second*10).Build(),
	)
	set, err := res.AsBool()
	if (err == nil || rueidis.IsRedisNil(err)) && !set {
		return ErrIdempotencyKeyExists
	}
	if err != nil {
		return fmt.Errorf("could not set idempotency key: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	key := r.connKey(envID)
	cmd := r.client.B().Hvals().Key(key).Build()

	res, err := r.client.Do(ctx, cmd).AsStrSlice()
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

func (r *redisConnectionStateManager) GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	key := r.connIndexByApp(envId, &appID)

	connIds, err := r.client.Do(ctx, r.client.B().Smembers().Key(key).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	if len(connIds) == 0 {
		return nil, nil
	}

	res, err := r.client.Do(ctx, r.client.B().Hmget().Key(r.connKey(envId)).Field(connIds...).Build()).AsStrSlice()
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
		r.connKey(envID),
		r.groupIDKey(envID, groupID),
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

func (r *redisConnectionStateManager) UpsertConnection(ctx context.Context, conn *Connection) error {
	envID, err := uuid.Parse(conn.Data.AuthData.GetEnvId())
	if err != nil {
		return fmt.Errorf("error parsing environment ID: %w", err)
	}

	groupID := conn.Group.Hash
	meta := &connpb.ConnMetadata{
		Id:              conn.Session.SessionId.ConnectionId,
		InstanceId:      conn.Session.SessionId.InstanceId,
		Status:          conn.Status,
		Language:        conn.Data.SdkLanguage,
		Version:         conn.Data.SdkVersion,
		GroupId:         groupID,
		Attributes:      conn.Data.SystemAttributes,
		GatewayId:       conn.GatewayId,
		LastHeartbeatAt: timestamppb.New(time.Now()),
	}

	isHealthy := "0"
	if conn.Status == connpb.ConnectionStatus_READY {
		isHealthy = "1"
	}

	keys := []string{
		r.connKey(envID),
		r.groupKey(envID),
		r.groupIDKey(envID, groupID),
		r.connIndexByApp(envID, conn.Group.AppID),
	}

	// NOTE: redis_state.StrSlice format the data in a non JSON way, not sure why
	var metaArg, groupArg string
	{
		byt, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("error serializing connection metadata: %w", err)
		}
		metaArg = string(byt)
	}

	{
		byt, err := json.Marshal(conn.Group)
		if err != nil {
			return fmt.Errorf("error serializing worker group data: %w", err)
		}
		groupArg = string(byt)
	}

	args := []string{
		meta.Id,
		metaArg,
		groupID,
		groupArg,
		isHealthy,
	}

	status, err := scripts["upsert_conn"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return err
	}

	switch status {
	case 0:
		return nil

	default:
		return fmt.Errorf("unknown status when storing connection metadata: %d", status)
	}
}

func (r *redisConnectionStateManager) DeleteConnection(ctx context.Context, envID uuid.UUID, appID *uuid.UUID, groupID string, connId string) error {
	keys := []string{
		r.connKey(envID),
		r.groupKey(envID),
		r.groupIDKey(envID, groupID),
		r.connIndexByApp(envID, appID),
	}

	args := []string{
		connId,
		groupID,
	}

	status, err := scripts["delete_conn"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting connection: %w", err)
	}

	switch status {
	case 0:
		return nil

	case 1:
		return ConnDeletedWithGroupErr

	default:
		return fmt.Errorf("unknow status when deleting connection: %w", err)
	}
}

func (r *redisConnectionStateManager) GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*WorkerGroup, error) {
	key := r.groupKey(envID)
	cmd := r.client.B().Hget().Key(key).Field(hash).Build()

	byt, err := r.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, WorkerGroupNotFoundErr
		}
		return nil, fmt.Errorf("error retrieving worker group: %w", err)
	}

	var group WorkerGroup
	if err := json.Unmarshal(byt, &group); err != nil {
		return nil, fmt.Errorf("error deserializing worker group: %w", err)
	}

	return &group, nil
}

func (r *redisConnectionStateManager) UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *WorkerGroup) error {
	byt, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("error serializing worker group for update: %w", err)
	}

	key := r.groupKey(envID)
	cmd := r.client.B().Hset().Key(key).FieldValue().FieldValue(group.Hash, string(byt)).Build()

	if err := r.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("error updating worker group: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) connKey(envID uuid.UUID) string {
	return fmt.Sprintf("{%s}:conns", envID)
}

func (r *redisConnectionStateManager) connIndexByApp(envID uuid.UUID, appId *uuid.UUID) string {
	// For the initial connection upsert, the app won't be synced just yet, so we cannot update this index.
	// We still need to provide a key with the same slot, otherwise Redis will complain
	if appId == nil || *appId == uuid.Nil {
		return fmt.Sprintf("{%s}:index_disabled", envID.String())
	}
	return fmt.Sprintf("{%s}:conns_appid:%s", envID.String(), appId.String())
}

func (r *redisConnectionStateManager) groupKey(envID uuid.UUID) string {
	return fmt.Sprintf("{%s}:groups", envID.String())
}

func (r *redisConnectionStateManager) groupIDKey(envID uuid.UUID, groupID string) string {
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
		r.client.B().Hset().Key(r.gatewaysHashKey()).FieldValue().FieldValue(gateway.Id, string(marshaled)).Build(),
	)
	if err := res.Error(); err != nil {
		return fmt.Errorf("could not set gateway state: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) DeleteGateway(ctx context.Context, gatewayId string) error {
	res := r.client.Do(
		ctx,
		r.client.B().Hdel().Key(r.gatewaysHashKey()).Field(gatewayId).Build(),
	)
	if err := res.Error(); err != nil {
		return fmt.Errorf("could not delete gateway state: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) GetGateway(ctx context.Context, gatewayId string) (*Gateway, error) {
	gatewayBytes, err := r.client.Do(
		ctx,
		r.client.B().Hget().Key(r.gatewaysHashKey()).Field(gatewayId).Build(),
	).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, GatewayNotFoundErr
		}

		return nil, fmt.Errorf("could not get gateway state: %w", err)
	}

	var gateway Gateway
	if err := json.Unmarshal(gatewayBytes, &gateway); err != nil {
		return nil, fmt.Errorf("could not unmarshal gateway state: %w", err)
	}

	return &gateway, nil
}
