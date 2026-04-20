import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { api, Aggregate } from "../api";

export const Route = createFileRoute("/compare")({
  component: Compare,
  validateSearch: (s: Record<string, unknown>) => ({
    a: (s.a as string) || "",
    b: (s.b as string) || "",
  }),
});

function Compare() {
  const { a, b } = Route.useSearch();
  const [data, setData] = useState<{
    a: Record<string, Aggregate>;
    b: Record<string, Aggregate>;
  } | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!a || !b) return;
    api.compare(a, b).then(setData).catch((e) => setErr(String(e)));
  }, [a, b]);

  if (!a || !b)
    return (
      <div className="container">
        <h1>Compare</h1>
        <p>Pick two runs from the history page to compare.</p>
      </div>
    );
  if (err) return <div className="container"><h1>Compare</h1><p>{err}</p></div>;
  if (!data) return <div className="container">Loading…</div>;

  const keys = Array.from(new Set([...Object.keys(data.a), ...Object.keys(data.b)]));
  return (
    <div className="container">
      <h1>Compare</h1>
      <div className="muted">
        A = <code>{a}</code>, B = <code>{b}</code>
      </div>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th>Metric</th>
              <th>A p50</th>
              <th>B p50</th>
              <th>Δ p50</th>
              <th>A p95</th>
              <th>B p95</th>
              <th>Δ p95</th>
              <th>A p99</th>
              <th>B p99</th>
              <th>Δ p99</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k) => {
              const av = data.a[k];
              const bv = data.b[k];
              return (
                <tr key={k}>
                  <td>{k}</td>
                  <td>{num(av?.p50)}</td>
                  <td>{num(bv?.p50)}</td>
                  <td>{delta(av?.p50, bv?.p50)}</td>
                  <td>{num(av?.p95)}</td>
                  <td>{num(bv?.p95)}</td>
                  <td>{delta(av?.p95, bv?.p95)}</td>
                  <td>{num(av?.p99)}</td>
                  <td>{num(bv?.p99)}</td>
                  <td>{delta(av?.p99, bv?.p99)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function num(v: number | undefined): string {
  if (v == null) return "—";
  return v.toFixed(2);
}
function delta(a?: number, b?: number): string {
  if (a == null || b == null) return "—";
  const d = b - a;
  const pct = a ? (d / a) * 100 : 0;
  const sign = d > 0 ? "+" : "";
  return `${sign}${d.toFixed(2)} (${sign}${pct.toFixed(1)}%)`;
}
