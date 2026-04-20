import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { api, RunConfig, Shape } from "../api";

export const Route = createFileRoute("/")({ component: Configure });

const NS = 1_000_000_000;

function Configure() {
  const nav = useNavigate();
  const [cfg, setCfg] = useState<RunConfig | null>(null);
  const [shapes, setShapes] = useState<Shape[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);

  useEffect(() => {
    api.defaults().then(setCfg).catch((e) => setErr(String(e)));
    api.shapes().then((s) => setShapes(s.shapes));
  }, []);

  if (!cfg) return <div className="container">Loading…</div>;

  const up = (patch: Partial<RunConfig>) => setCfg({ ...cfg, ...patch });

  return (
    <div className="container">
      <h1>Configure run</h1>
      {err && <div className="card" style={{ color: "crimson" }}>{err}</div>}

      <div className="card">
        <h2>Target</h2>
        <div className="row">
          <label>
            Mode
            <select
              value={cfg.target.mode ?? "dev"}
              onChange={(e) =>
                up({
                  target: { ...cfg.target, mode: e.target.value as "dev" | "selfhosted" },
                })
              }
            >
              <option value="dev">Dev server (inngest dev)</option>
              <option value="selfhosted">Self-hosted (inngest start)</option>
            </select>
          </label>
          <label>
            URL
            <input
              value={cfg.target.url}
              onChange={(e) => up({ target: { ...cfg.target, url: e.target.value } })}
            />
          </label>
          <label>
            Event key{cfg.target.mode === "selfhosted" ? "" : " (optional)"}
            <input
              value={cfg.target.eventKey ?? ""}
              onChange={(e) =>
                up({ target: { ...cfg.target, eventKey: e.target.value || null } })
              }
            />
          </label>
          <label>
            Signing key{cfg.target.mode === "selfhosted" ? " (required)" : " (optional)"}
            <input
              value={cfg.target.signingKey ?? ""}
              onChange={(e) =>
                up({ target: { ...cfg.target, signingKey: e.target.value || null } })
              }
            />
          </label>
        </div>
        {cfg.target.mode === "selfhosted" && (
          <div className="muted" style={{ marginTop: 8 }}>
            Self-hosted mode runs the SDK with strict signature verification.
            The signing key must match <code>inngest start --signing-key</code>.
          </div>
        )}
      </div>

      <div className="card">
        <h2>Load</h2>
        <div className="row">
          <label>
            Apps
            <input
              type="number"
              min={1}
              value={cfg.apps}
              onChange={(e) => up({ apps: +e.target.value })}
            />
          </label>
          <label>
            Functions / app
            <input
              type="number"
              min={1}
              value={cfg.functionsPerApp}
              onChange={(e) => up({ functionsPerApp: +e.target.value })}
            />
          </label>
          <label>
            Concurrency (firer workers)
            <input
              type="number"
              min={1}
              value={cfg.concurrency}
              onChange={(e) => up({ concurrency: +e.target.value })}
            />
          </label>
          <label>
            Event rate (per sec)
            <input
              type="number"
              min={1}
              value={cfg.eventRate}
              onChange={(e) => up({ eventRate: +e.target.value })}
            />
          </label>
          <label>
            Batch size
            <input
              type="number"
              min={1}
              value={cfg.batchSize}
              onChange={(e) => up({ batchSize: +e.target.value })}
            />
          </label>
          <label>
            Duration (s)
            <input
              type="number"
              min={1}
              value={cfg.duration / NS}
              onChange={(e) => up({ duration: +e.target.value * NS })}
            />
          </label>
          <label>
            Warmup (s)
            <input
              type="number"
              min={0}
              value={cfg.warmup / NS}
              onChange={(e) => up({ warmup: +e.target.value * NS })}
            />
          </label>
        </div>
      </div>

      <div className="card">
        <h2>Shape mix (weights)</h2>
        <div className="grid">
          {shapes.map((s) => (
            <label key={s}>
              {s}
              <input
                type="number"
                min={0}
                value={cfg.shapeMix[s] ?? 0}
                onChange={(e) =>
                  up({ shapeMix: { ...cfg.shapeMix, [s]: +e.target.value } })
                }
              />
            </label>
          ))}
        </div>
      </div>

      <div className="row">
        <button
          className="primary"
          disabled={starting}
          onClick={async () => {
            setErr(null);
            setStarting(true);
            try {
              const { id } = await api.startRun(cfg);
              nav({ to: "/runs/$id", params: { id } });
            } catch (e: unknown) {
              setErr(String(e));
              setStarting(false);
            }
          }}
        >
          {starting ? "Starting…" : "Start run"}
        </button>
        <span className="muted">
          Target must be a running Inngest server (dev or otherwise). Miniredis-backed dev
          mode is not representative of production latencies.
        </span>
      </div>
    </div>
  );
}
