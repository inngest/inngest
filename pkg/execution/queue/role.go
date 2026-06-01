package queue

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
)

const (
	QueueRoleSequential      = "sequential"
	QueueRoleScavenger       = "scavenger"
	QueueRoleInstrumentation = "instrument"
	QueueRoleLatencyTracker  = "latency"
)

// QueueRole is a leased background responsibility for a queue processor.
// Each role leases its Name() per shard before Run is called at RunInterval().
type QueueRole interface {
	// Name returns the shard-scoped lease key for this role.
	Name() string

	// LeaseDuration returns how long the role lease is held before renewal.
	LeaseDuration() time.Duration

	// RunInterval returns how often Run should be called while this worker holds the role lease.
	RunInterval() time.Duration

	// ExcludesScanning reports whether holding this role should pause normal queue scanning.
	ExcludesScanning() bool

	// Run performs the role's periodic work against the leased shard.
	Run(ctx context.Context, shard QueueShard) error
}

type QueueRoleStatus struct {
	Name             string
	LeaseID          ulid.ULID
	LeaseExpiresAt   time.Time
	ExcludesScanning bool
}

type QueueRoleOpt func(*queueRole)

// WithRoleExcludesScanning makes a role suppress normal queue scanning while
// this worker actively holds that role's lease.
func WithRoleExcludesScanning(exclude bool) QueueRoleOpt {
	return func(r *queueRole) {
		r.excludesScanning = exclude
	}
}

// WithRoleRunInterval overrides the role callback interval.
func WithRoleRunInterval(interval time.Duration) QueueRoleOpt {
	return func(r *queueRole) {
		if interval > 0 {
			r.runInterval = interval
		}
	}
}

type queueRole struct {
	name             string
	leaseDuration    time.Duration
	runInterval      time.Duration
	excludesScanning bool
	run              func(context.Context, QueueShard) error
}

func (r queueRole) Name() string {
	return r.name
}

func (r queueRole) LeaseDuration() time.Duration {
	return r.leaseDuration
}

func (r queueRole) RunInterval() time.Duration {
	return r.runInterval
}

func (r queueRole) ExcludesScanning() bool {
	return r.excludesScanning
}

func (r queueRole) Run(ctx context.Context, shard QueueShard) error {
	if r.run == nil {
		return nil
	}
	return r.run(ctx, shard)
}

func newQueueRole(
	name string,
	leaseDuration time.Duration,
	runInterval time.Duration,
	run func(context.Context, QueueShard) error,
	opts ...QueueRoleOpt,
) queueRole {
	role := queueRole{
		name:          name,
		leaseDuration: leaseDuration,
		runInterval:   runInterval,
		run:           run,
	}
	for _, opt := range opts {
		opt(&role)
	}
	return role
}

func (q *queueProcessor) runRole(ctx context.Context, role QueueRole) {
	if role == nil {
		return
	}
	if role.Name() == QueueRoleSequential && len(q.AllowQueues) > 0 {
		return
	}

	name := role.Name()
	if name == "" {
		q.quit <- fmt.Errorf("queue role name cannot be empty")
		return
	}

	leaseDuration := role.LeaseDuration()
	if leaseDuration <= 0 {
		q.quit <- fmt.Errorf("queue role %q must define a positive lease duration", name)
		return
	}

	shard := q.Shard()
	claim := func(initial bool) bool {
		leaseID, err := shard.RoleLease(ctx, name, leaseDuration, q.roleLease(name))
		if err == ErrRoleAlreadyLeased {
			q.setRoleLease(ctx, name, nil, shard)
			return true
		}
		if err != nil {
			q.setRoleLease(ctx, name, nil, shard)
			if initial {
				q.quit <- err
				return false
			}
			logger.StdlibLogger(ctx).Error("error claiming queue role lease", "role", name, "error", err)
			return true
		}
		q.setRoleLease(ctx, name, leaseID, shard)
		return true
	}

	if !claim(true) {
		return
	}

	leaseTick := q.Clock().NewTicker(leaseDuration / 3)
	defer leaseTick.Stop()

	var runTick clockwork.Ticker
	var runC <-chan time.Time
	if role.RunInterval() > 0 {
		runTick = q.Clock().NewTicker(role.RunInterval())
		defer runTick.Stop()
		runC = runTick.Chan()
	}

	for {
		select {
		case <-ctx.Done():
			q.setRoleLease(context.Background(), name, nil, shard)
			return
		case <-leaseTick.Chan():
			claim(false)
		case <-runC:
			if q.isRoleActive(name) {
				if err := role.Run(ctx, shard); err != nil {
					logger.StdlibLogger(ctx).Error("error running queue role", "role", name, "error", err)
				}
			}
		}
	}
}

func (q *queueProcessor) roleLease(roleName string) *ulid.ULID {
	q.roleLeaseLock.RLock()
	defer q.roleLeaseLock.RUnlock()

	leaseID := q.roleLeaseIDs[roleName]
	if leaseID == nil {
		return nil
	}
	copied := *leaseID
	return &copied
}

func (q *queueProcessor) setRoleLease(ctx context.Context, roleName string, leaseID *ulid.ULID, shard QueueShard) {
	q.roleLeaseLock.Lock()
	defer q.roleLeaseLock.Unlock()

	previous := q.roleLeaseIDs[roleName]
	previousActive := previous != nil && ulid.Time(previous.Time()).After(q.Clock().Now())
	nextActive := leaseID != nil && ulid.Time(leaseID.Time()).After(q.Clock().Now())

	if !previousActive && nextActive {
		switch roleName {
		case QueueRoleSequential:
			metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		case QueueRoleInstrumentation:
			logger.StdlibLogger(ctx).Debug("claimed instrumentation lease")
			metrics.IncrInstrumentationLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		}
	}
	if previousActive && !nextActive && roleName == QueueRoleInstrumentation {
		logger.StdlibLogger(ctx).Debug("lost instrumentation lease")
	}

	q.roleLeaseIDs[roleName] = leaseID
}

func (q *queueProcessor) isRoleActive(roleName string) bool {
	l := q.roleLease(roleName)
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock().Now())
}

func (q *queueProcessor) isSequential() bool {
	return q.isRoleActive(QueueRoleSequential)
}

func (q *queueProcessor) scanningExcludedByRole() string {
	q.roleLeaseLock.RLock()
	defer q.roleLeaseLock.RUnlock()

	now := q.Clock().Now()
	for _, role := range q.roles {
		if role == nil || !role.ExcludesScanning() {
			continue
		}
		leaseID := q.roleLeaseIDs[role.Name()]
		if leaseID != nil && ulid.Time(leaseID.Time()).After(now) {
			return role.Name()
		}
	}
	return ""
}

func (q *queueProcessor) ActiveRoles() []QueueRoleStatus {
	if q.roleLeaseLock == nil {
		return nil
	}

	q.roleLeaseLock.RLock()
	defer q.roleLeaseLock.RUnlock()

	now := q.Clock().Now()
	statuses := []QueueRoleStatus{}
	seen := map[string]struct{}{}
	for _, role := range q.roles {
		if role == nil {
			continue
		}
		seen[role.Name()] = struct{}{}
		if status, ok := q.activeRoleStatusLocked(role.Name(), role.ExcludesScanning(), now); ok {
			statuses = append(statuses, status)
		}
	}

	extraNames := []string{}
	for name := range q.roleLeaseIDs {
		if _, ok := seen[name]; ok {
			continue
		}
		extraNames = append(extraNames, name)
	}
	sort.Strings(extraNames)
	for _, name := range extraNames {
		if status, ok := q.activeRoleStatusLocked(name, false, now); ok {
			statuses = append(statuses, status)
		}
	}

	return statuses
}

func (q *queueProcessor) activeRoleStatusLocked(roleName string, excludesScanning bool, now time.Time) (QueueRoleStatus, bool) {
	leaseID := q.roleLeaseIDs[roleName]
	if leaseID == nil {
		return QueueRoleStatus{}, false
	}

	expiresAt := ulid.Time(leaseID.Time())
	if !expiresAt.After(now) {
		return QueueRoleStatus{}, false
	}

	return QueueRoleStatus{
		Name:             roleName,
		LeaseID:          *leaseID,
		LeaseExpiresAt:   expiresAt,
		ExcludesScanning: excludesScanning,
	}, true
}
