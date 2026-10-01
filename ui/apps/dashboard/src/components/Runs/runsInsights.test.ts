import { describe, expect, it } from 'vitest';

import { runsInsightsHref } from './runsInsights';

describe('runsInsightsHref', () => {
  it('preserves CEL and bounds the ClickHouse search to three days', () => {
    const href = runsInsightsHref({
      envSlug: 'prod/us west',
      celQuery:
        "event.data.customer == 'José & Sons'\n(output.total >= 42 || output.vip == true)",
      functionSlug: null,
    });

    const url = new URL(href!, 'https://app.inngest.com');
    expect(url.pathname).toBe('/env/prod%2Fus%20west/insights');
    expect(url.searchParams.get('name')).toBe('Runs search');
    expect(url.searchParams.get('sql')).toBe(`SELECT
    *
FROM
    runs
WHERE
    cel(\`(event.data.customer == 'José & Sons') &&
((output.total >= 42 || output.vip == true))\`)
    AND queued_at > now() - INTERVAL 3 DAY`);
    expect([...url.searchParams.keys()].sort()).toEqual(['name', 'sql']);
  });

  it('uses the fully qualified function slug while omitting unsupported filters', () => {
    const href = runsInsightsHref({
      envSlug: 'staging',
      celQuery: "error.code != 'E_STOP'",
      functionSlug: "billing-app-worker's-charge-invoice",
    });

    const url = new URL(href!, 'https://app.inngest.com');
    const sql = url.searchParams.get('sql');
    expect(sql).toContain(
      "function_id = 'billing-app-worker\\'s-charge-invoice'",
    );
    expect(sql).not.toContain('app_id');
    expect([...url.searchParams.keys()].sort()).toEqual(['name', 'sql']);
    expect(href).not.toContain('status=');
    expect(href).not.toContain('isDeferred=');
    expect(href).not.toContain('from=');
    expect(href).not.toContain('until=');
  });

  it('omits the handoff when CEL contains an unrepresentable backtick', () => {
    expect(
      runsInsightsHref({
        envSlug: 'production',
        celQuery: "event.data[`customer`] == 'A'",
        functionSlug: null,
      }),
    ).toBeUndefined();
  });

  it('omits the handoff when generated SQL exceeds the 8 KiB route contract', () => {
    expect(
      runsInsightsHref({
        envSlug: 'production',
        celQuery: `event.data.value == "${'a'.repeat(8 * 1024)}"`,
        functionSlug: null,
      }),
    ).toBeUndefined();
  });
});
