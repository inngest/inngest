import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { api, RunRow } from "../api";

export const Route = createFileRoute("/history")({ component: History });

function History() {
  const [runs, setRuns] = useState<RunRow[]>([]);
  const [sel, setSel] = useState<[string | null, string | null]>([null, null]);

  useEffect(() => {
    api.listRuns().then((d) => setRuns(d.runs));
  }, []);

  const toggle = (id: string) => {
    const [a, b] = sel;
    if (a === id) return setSel([b, null]);
    if (b === id) return setSel([a, null]);
    if (!a) return setSel([id, b]);
    if (!b) return setSel([a, id]);
    setSel([id, b]);
  };

  return (
    <div className="container">
      <h1>History</h1>
      <div className="row">
        <Link to="/compare" search={{ a: sel[0] ?? "", b: sel[1] ?? "" }}>
          <button disabled={!sel[0] || !sel[1]}>
            Compare selected{sel[0] && sel[1] ? "" : " (pick 2)"}
          </button>
        </Link>
      </div>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th></th>
              <th>ID</th>
              <th>Name</th>
              <th>Status</th>
              <th>Started</th>
              <th>Duration</th>
              <th>Dropped</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {runs.map((r) => {
              const dur =
                r.endedAt && r.createdAt
                  ? Math.round((r.endedAt - r.createdAt) / 1e9) + "s"
                  : "—";
              return (
                <tr key={r.id}>
                  <td>
                    <input
                      type="checkbox"
                      checked={sel[0] === r.id || sel[1] === r.id}
                      onChange={() => toggle(r.id)}
                    />
                  </td>
                  <td><code>{r.id}</code></td>
                  <td>{r.config?.name ?? "—"}</td>
                  <td className={"status-" + r.status}>{r.status}</td>
                  <td>{new Date(r.createdAt / 1e6).toLocaleString()}</td>
                  <td>{dur}</td>
                  <td>{r.samplesDropped}</td>
                  <td>
                    <Link to="/runs/$id" params={{ id: r.id }}>view</Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
