import { describe, expect, it } from 'vitest';

import { formatSQL } from '@/components/Insights/InsightsSQLEditor/utils';
import { RunsOrderByField } from '@/gql/graphql';
import { runsInsightsHref } from './runsInsights';

describe('runsInsightsHref', () => {
  it('encodes CEL, the selected time field and range, and external app slugs in SQL', () => {
    const href = runsInsightsHref({
      envSlug: 'prod/us west',
      celQuery: "event.data.customer == 'José & Sons'\n|| output.total >= 42",
      startTime: '2026-02-03T04:05:06.789Z',
      endTime: '2026-02-04T07:08:09.123Z',
      timeField: RunsOrderByField.StartedAt,
      appIDs: ['payments-api', "partner's-api"],
      functionSlug: null,
    });

    const url = new URL(href!, 'https://app.inngest.com');
    expect(url.pathname).toBe('/env/prod%2Fus%20west/insights');
    expect(url.searchParams.get('name')).toBe('Runs search');
    const sql = url.searchParams.get('sql');
    expect(sql).toBe(`SELECT
    *
FROM
    runs
WHERE
    cel(\`event.data.customer == 'José & Sons'
|| output.total >= 42\`)
    AND started_at >= '2026-02-03 04:05:06.789'
    AND started_at < '2026-02-04 07:08:09.123'
    AND app_id IN ('payments-api', 'partner\\'s-api')
ORDER BY
    started_at DESC
LIMIT 1000`);
    expect(formatSQL(sql!)).toContain(
      "`event.data.customer == 'José & Sons'\n|| output.total >= 42`",
    );
    expect([...url.searchParams.keys()].sort()).toEqual(['name', 'sql']);
  });

  it('uses fully qualified function scope instead of app filters and omits unsupported filters', () => {
    const href = runsInsightsHref({
      envSlug: 'staging',
      celQuery: "error.code != 'E_STOP'",
      startTime: '2026-07-08T09:10:11Z',
      endTime: null,
      timeField: RunsOrderByField.EndedAt,
      appIDs: ['ignored-app'],
      functionSlug: 'billing-worker-charge-invoice',
    });

    const url = new URL(href!, 'https://app.inngest.com');
    const sql = url.searchParams.get('sql');
    expect(sql).toContain("ended_at >= '2026-07-08 09:10:11.000'");
    expect(sql).toContain("function_id = 'billing-worker-charge-invoice'");
    expect(sql).not.toContain('app_id');
    expect(href).not.toContain('cel=');
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
        startTime: '2026-01-01T00:00:00Z',
        endTime: null,
        timeField: RunsOrderByField.QueuedAt,
        appIDs: null,
        functionSlug: null,
      }),
    ).toBeUndefined();
  });

  it('omits the handoff when generated SQL exceeds the Insights URL contract', () => {
    expect(
      runsInsightsHref({
        envSlug: 'production',
        celQuery: 'event.data.ok == true',
        startTime: '2026-01-01T00:00:00Z',
        endTime: null,
        timeField: RunsOrderByField.QueuedAt,
        appIDs: ['app-'.repeat(2200)],
        functionSlug: null,
      }),
    ).toBeUndefined();
  });
});
