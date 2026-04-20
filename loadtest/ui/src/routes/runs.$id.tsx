import { createFileRoute, useParams } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { api, Aggregate, LiveStats, LogEntry, RunRow } from "../api";

export const Route = createFileRoute("/runs/$id")({ component: RunView });

interface Series {
  ts: number;
  count: number;
}

function RunView() {
  const { id } = useParams({ from: "/runs/$id" });
  const [run, setRun] = useState<RunRow | null>(null);
  const [aggs, setAggs] = useState<Record<string, Aggregate>>({});
  const [stats, setStats] = useState<LiveStats | null>(null);
  const [series, setSeries] = useState<Series[]>([]);
  const cursor = useRef(0);

  useEffect(() => {
    let stopped = false;
    const tick = async () => {
      try {
        const runRow = await api.getRun(id);
        if (stopped) return;
        setRun(runRow);
        if (runRow.status !== "pending") {
          try {
            setAggs(await api.aggregates(id));
          } catch {}
        }
        const live = await api.liveSamples(id, cursor.current);
        if (stopped) return;
        cursor.current = live.cursor;
        if (live.stats) setStats(coerceStats(live.stats));
        if (live.samples.length) {
          const bucket = new Map<number, number>();
          for (const s of live.samples) {
            if (s.phase !== "fn_start") continue;
            const sec = Math.floor(s.ts / 1e9);
            bucket.set(sec, (bucket.get(sec) ?? 0) + 1);
          }
          setSeries((prev) => {
            const next = [...prev];
            for (const [sec, count] of bucket) {
              const idx = next.findIndex((p) => p.ts === sec);
              if (idx >= 0) next[idx] = { ...next[idx], count: next[idx].count + count };
              else next.push({ ts: sec, count });
            }
            next.sort((a, b) => a.ts - b.ts);
            if (next.length > 300) next.splice(0, next.length - 300);
            return next;
          });
        }
      } catch (e) {
        console.error(e);
      }
    };
    tick();
    const h = setInterval(tick, 1500);
    return () => {
      stopped = true;
      clearInterval(h);
    };
  }, [id]);

  if (!run) return <div className="container">Loading…</div>;

  return (
    <div className="container">
      <h1>
        Run <code>{id}</code>{" "}
        <span className={"status-" + run.status}>{run.status}</span>
      </h1>
      <div className="row">
        <span className="muted">
          started {new Date(run.createdAt / 1e6).toLocaleString()}
          {run.endedAt && <> · ended {new Date(run.endedAt / 1e6).toLocaleString()}</>}
        </span>
        {run.status === "running" && (
          <button className="danger" onClick={() => api.stopRun(id)}>
            Stop
          </button>
        )}
      </div>

      {topLevelError(run, stats) && (
        <div
          className="card"
          style={{ borderColor: "crimson", borderWidth: 2 }}
        >
          <h2 style={{ color: "crimson" }}>Error</h2>
          <div style={{ whiteSpace: "pre-wrap", fontFamily: "ui-monospace, monospace", fontSize: 13 }}>
            {topLevelError(run, stats)}
          </div>
        </div>
      )}

      <div className="card">
        <h2>Summary</h2>
        <div className="grid">
          <StatRow label="Events fired" value={stats?.eventsFired} />
          <StatRow label="Events failed" value={stats?.eventsFailed} danger={!!stats?.eventsFailed} />
          <StatRow label="Functions started" value={stats?.functionsStarted} />
          <StatRow label="Functions completed" value={stats?.functionsEnded} />
          <StatRow label="Steps started" value={stats?.stepsStarted} />
          <StatRow label="Steps completed" value={stats?.stepsEnded} />
          <StatRow label="Samples observed" value={stats?.samplesObserved} />
          <StatRow label="Workers alive" value={stats?.workersAlive} />
          <StatRow
            label="Workers failed"
            value={stats?.workersFailed}
            danger={!!stats?.workersFailed}
          />
        </div>
        {stats?.lastFireError && (
          <div className="muted" style={{ color: "crimson", marginTop: 8 }}>
            Last fire error: {stats.lastFireError}
          </div>
        )}
      </div>

      {stats?.log && stats.log.length > 0 && (
        <div className="card">
          <h2>Run log</h2>
          <LogView entries={stats.log} />
        </div>
      )}

      {workerStderr(run, stats) && (
        <div className="card">
          <h2>Worker stderr</h2>
          <WorkerStderrView tails={workerStderr(run, stats)!} />
        </div>
      )}

      <div className="card">
        <h2>Throughput (fn starts / sec)</h2>
        <Sparkline points={series.map((s) => s.count)} />
      </div>

      <div className="grid">
        {["event_to_run_ms", "inter_step_ms", "step_duration_ms", "sdk_overhead_ms"].map(
          (k) => (
            <div className="card" key={k}>
              <h2>{prettyMetric(k)}</h2>
              <div className="metric">
                <span>p50</span>
                <span>{fmtMs(aggs[k]?.p50)}</span>
              </div>
              <div className="metric">
                <span>p95</span>
                <span>{fmtMs(aggs[k]?.p95)}</span>
              </div>
              <div className="metric">
                <span>p99</span>
                <span>{fmtMs(aggs[k]?.p99)}</span>
              </div>
              <div className="metric">
                <span>samples</span>
                <span>{aggs[k]?.count ?? 0}</span>
              </div>
              {k === "inter_step_ms" && (
                <div className="muted">
                  includes checkpoint + queue + dispatch
                </div>
              )}
            </div>
          )
        )}
      </div>
    </div>
  );
}

function StatRow({
  label,
  value,
  danger,
}: {
  label: string;
  value: number | undefined;
  danger?: boolean;
}) {
  return (
    <div className="metric">
      <span>{label}</span>
      <span style={danger ? { color: "crimson", fontWeight: 600 } : undefined}>
        {value ?? 0}
      </span>
    </div>
  );
}

function topLevelError(run: RunRow, stats: LiveStats | null): string | null {
  const pieces: string[] = [];
  const summaryErr = run.summary && (run.summary as Record<string, unknown>)["error"];
  if (typeof summaryErr === "string" && summaryErr) pieces.push(summaryErr);
  if (stats?.lastFireError && !pieces.some((p) => p.includes(stats.lastFireError!))) {
    pieces.push("firer: " + stats.lastFireError);
  }
  return pieces.length ? pieces.join("\n\n") : null;
}

function workerStderr(run: RunRow, stats: LiveStats | null): Record<string, string> | null {
  if (stats?.workerStderr && Object.keys(stats.workerStderr).length) return stats.workerStderr;
  const fromSummary = run.summary && (run.summary as Record<string, unknown>)["workerStderr"];
  if (fromSummary && typeof fromSummary === "object") return fromSummary as Record<string, string>;
  return null;
}

function LogView({ entries }: { entries: LogEntry[] }) {
  return (
    <pre
      style={{
        whiteSpace: "pre-wrap",
        fontSize: 12,
        fontFamily: "ui-monospace, monospace",
        maxHeight: 260,
        overflow: "auto",
        margin: 0,
      }}
    >
      {entries.map((e, i) => {
        const t = new Date(e.ts / 1e6).toLocaleTimeString();
        const color =
          e.level === "error" ? "crimson" : e.level === "warn" ? "#a16207" : undefined;
        return (
          <div key={i} style={{ color }}>
            {t} [{e.level}] {e.msg}
          </div>
        );
      })}
    </pre>
  );
}

function WorkerStderrView({ tails }: { tails: Record<string, string> }) {
  return (
    <>
      {Object.entries(tails).map(([id, body]) => (
        <details key={id} open>
          <summary style={{ cursor: "pointer", marginBottom: 4 }}>
            <code>{id}</code>
          </summary>
          <pre
            style={{
              whiteSpace: "pre-wrap",
              fontSize: 12,
              fontFamily: "ui-monospace, monospace",
              maxHeight: 240,
              overflow: "auto",
              background: "rgba(127,127,127,0.08)",
              padding: 8,
              borderRadius: 4,
            }}
          >
            {body}
          </pre>
        </details>
      ))}
    </>
  );
}

// The live stats blob can arrive as the typed LiveStats (while the run is
// active) or as a persisted summary map (once the run ends). Normalize.
function coerceStats(raw: LiveStats | Record<string, unknown>): LiveStats {
  const r = raw as Record<string, unknown>;
  const n = (k: string) => Number(r[k] ?? 0);
  const s = (k: string) => String(r[k] ?? "");
  const logRaw = Array.isArray(r["log"]) ? (r["log"] as LogEntry[]) : undefined;
  const stderrRaw =
    r["workerStderr"] && typeof r["workerStderr"] === "object"
      ? (r["workerStderr"] as Record<string, string>)
      : undefined;
  return {
    eventsFired: n("eventsFired"),
    eventsFailed: n("eventsFailed"),
    lastFireError: s("lastFireError") || undefined,
    functionsStarted: n("functionsStarted"),
    functionsEnded: n("functionsEnded"),
    stepsStarted: n("stepsStarted"),
    stepsEnded: n("stepsEnded"),
    samplesObserved: n("samplesObserved"),
    workersReady: n("workersReady"),
    workersAlive: n("workersAlive"),
    workersFailed: n("workersFailed"),
    log: logRaw,
    workerStderr: stderrRaw,
  };
}

function prettyMetric(k: string) {
  return k
    .replace(/_ms$/, " (ms)")
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function fmtMs(v: number | undefined): string {
  if (v == null) return "—";
  if (v < 1) return v.toFixed(2);
  if (v < 10) return v.toFixed(1);
  return Math.round(v).toString();
}

function Sparkline({ points }: { points: number[] }) {
  if (!points.length) return <div className="muted">waiting for samples…</div>;
  const max = Math.max(1, ...points);
  const w = 600;
  const h = 80;
  const path = points
    .map((v, i) => {
      const x = (i / Math.max(1, points.length - 1)) * w;
      const y = h - (v / max) * h;
      return `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <svg viewBox={`0 0 ${w} ${h}`} width="100%" height={h}>
      <path d={path} stroke="#2563eb" strokeWidth={1.5} fill="none" />
    </svg>
  );
}
