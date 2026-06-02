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

	// OnLeaseTick performs processor-local work before each lease renewal.
	OnLeaseTick(ctx context.Context, shard QueueShard)
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
	onLeaseTick      func(context.Context, QueueShard)
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

func (r queueRole) OnLeaseTick(ctx context.Context, shard QueueShard) {
	if r.onLeaseTick != nil {
		r.onLeaseTick(ctx, shard)
	}
}

func newQueueRole(
	name string,
	leaseDuration time.Duration,
	runInterval time.Duration,
	run func(context.Context, QueueShard) error,
	onLeaseTick func(context.Context, QueueShard),
	opts ...QueueRoleOpt,
) queueRole {
	role := queueRole{
		name:          name,
		leaseDuration: leaseDuration,
		runInterval:   runInterval,
		run:           run,
		onLeaseTick:   onLeaseTick,
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

	// claim attempts to acquire or renew the role lease.  The current lease ID
	// is passed back to the shard so the same worker can renew leases it already
	// owns, while another active owner's lease is treated as expected contention.
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

	// The first claim decides whether this role goroutine can start.  A startup
	// lease error means queue startup should fail instead of leaving a broken
	// background loop running.
	if !claim(true) {
		return
	}

	// leaseTick renews the ownership lease before it expires.  This is separate
	// from role execution so a role can keep ownership even if RunInterval is
	// longer than the lease duration.
	leaseTick := q.Clock().NewTicker(leaseDuration / 3)
	defer leaseTick.Stop()

	var runTick clockwork.Ticker
	var runC <-chan time.Time
	if role.RunInterval() > 0 {
		// runTick controls how often the role's actual work runs.  If the
		// interval is zero, runC stays nil and this select case is disabled.
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
			role.OnLeaseTick(ctx, shard)
			claim(false)
		case <-runC:
			// The worker may have lost the lease on a renewal tick; only run
			// role work while this process still owns an active lease.
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

	// Treat active state as a transition, not merely nil/non-nil.  A stored ULID
	// can be expired, and renewing an already-active lease should not emit
	// claim metrics again.
	previousActive := previous != nil && ulid.Time(previous.Time()).After(q.Clock().Now())
	nextActive := leaseID != nil && ulid.Time(leaseID.Time()).After(q.Clock().Now())

	if !previousActive && nextActive {
		logger.StdlibLogger(ctx).Debug(
			"acquired queue role lease",
			"role", roleName,
			"queue_shard", shard.Name(),
			"lease_expires_at", ulid.Time(leaseID.Time()),
		)

		switch roleName {
		case QueueRoleSequential:
			metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		case QueueRoleInstrumentation:
			metrics.IncrInstrumentationLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		}
	}

	if previousActive && !nextActive {
		logger.StdlibLogger(ctx).Debug(
			"lost queue role lease",
			"role", roleName,
			"queue_shard", shard.Name(),
		)
	}

	// Store the latest lease value, including nil when contention or errors mean
	// this worker does not currently own the role.
	q.roleLeaseIDs[roleName] = leaseID
}

func (q *queueProcessor) isRoleActive(roleName string) bool {
	l := q.roleLease(roleName)
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock().Now())
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
