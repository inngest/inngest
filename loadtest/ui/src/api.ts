export type Shape =
  | "noop"
  | "steps-3"
  | "steps-10"
  | "sleep-1s"
  | "fanout-5"
  | "retry-forced";

export type TargetMode = "dev" | "selfhosted";

export interface Target {
  url: string;
  mode?: TargetMode;
  eventKey?: string | null;
  signingKey?: string | null;
}

export interface RunConfig {
  name: string;
  target: Target;
  apps: number;
  functionsPerApp: number;
  shapeMix: Record<string, number>;
  concurrency: number;
  eventRate: number;
  duration: number; // nanoseconds (Go time.Duration serializes as int64 ns)
  warmup: number;
  batchSize: number;
}

export interface RunRow {
  id: string;
  createdAt: number;
  endedAt?: number;
  status: "pending" | "running" | "completed" | "failed" | "stopped";
  config: RunConfig;
  summary?: Record<string, unknown>;
  samplesDropped: number;
}

export interface Aggregate {
  p50: number;
  p95: number;
  p99: number;
  count: number;
}

export interface LiveSample {
  workerId: string;
  phase: string;
  fn: string;
  step: string;
  ts: number;
  runId?: string;
}

export interface LogEntry {
  ts: number;
  level: "info" | "warn" | "error" | string;
  msg: string;
}

export interface LiveStats {
  status?: string;
  eventsFired: number;
  eventsFailed: number;
  lastFireError?: string;
  functionsStarted: number;
  functionsEnded: number;
  stepsStarted: number;
  stepsEnded: number;
  samplesObserved: number;
  workersReady: number;
  workersAlive: number;
  workersFailed: number;
  log?: LogEntry[];
  workerStderr?: Record<string, string>;
}

async function json<T>(r: Response): Promise<T> {
  if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText);
  return r.json();
}

export const api = {
  defaults: () => fetch("/api/defaults").then((r) => json<RunConfig>(r)),
  shapes: () => fetch("/api/shapes").then((r) => json<{ shapes: Shape[] }>(r)),
  listRuns: () =>
    fetch("/api/runs").then((r) => json<{ runs: RunRow[] }>(r)),
  getRun: (id: string) => fetch(`/api/runs/${id}`).then((r) => json<RunRow>(r)),
  startRun: (cfg: RunConfig) =>
    fetch("/api/runs", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(cfg),
    }).then((r) => json<{ id: string }>(r)),
  stopRun: (id: string) =>
    fetch(`/api/runs/${id}/stop`, { method: "POST" }).then((r) =>
      json<{ status: string }>(r)
    ),
  liveSamples: (id: string, after: number) =>
    fetch(`/api/runs/${id}/live?after=${after}`).then((r) =>
      json<{
        samples: LiveSample[];
        cursor: number;
        stats: LiveStats | Record<string, unknown> | null;
      }>(r)
    ),
  aggregates: (id: string) =>
    fetch(`/api/runs/${id}/aggregates`).then((r) => json<Record<string, Aggregate>>(r)),
  compare: (a: string, b: string) =>
    fetch(`/api/runs/compare?a=${a}&b=${b}`).then((r) =>
      json<{ a: Record<string, Aggregate>; b: Record<string, Aggregate> }>(r)
    ),
};
