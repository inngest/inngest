import { describe, expect, it } from 'vitest';

import {
  consumeSQLPrefillURL,
  MAX_INSIGHTS_NAME_BYTES,
  MAX_INSIGHTS_QUERY_ID_BYTES,
  MAX_INSIGHTS_SQL_BYTES,
  savedQueryInsightsURL,
  sqlPrefillInsightsURL,
  syncSavedQueryURL,
  validateInsightsSearch,
} from './insightsSearchParams';

describe('validateInsightsSearch', () => {
  it('accepts one saved-query mode and discards SQL-prefill fields', () => {
    expect(
      validateInsightsSearch({
        query_id: '  saved-query-17  ',
        sql: "SELECT 'must-not-survive'",
        name: 'also discarded',
      }),
    ).toEqual({
      intent: { kind: 'saved-query', id: 'saved-query-17' },
    });
  });

  it('accepts SQL-prefill mode with an optional name', () => {
    const sql = "SELECT 'space & plus + slash /' AS marker";

    expect(validateInsightsSearch({ sql, name: '  Failed: café  ' })).toEqual({
      intent: {
        kind: 'sql-prefill',
        sql,
        name: 'Failed: café',
      },
    });
  });

  it('rejects blank and oversized SQL', () => {
    expect(validateInsightsSearch({ sql: ' \n\t ' })).toEqual({});
    expect(
      validateInsightsSearch({
        sql: 'é'.repeat(MAX_INSIGHTS_SQL_BYTES / 2 + 1),
      }),
    ).toEqual({});
  });

  it('omits an oversized name without discarding valid SQL', () => {
    expect(
      validateInsightsSearch({
        sql: 'SELECT 17',
        name: 'é'.repeat(MAX_INSIGHTS_NAME_BYTES / 2 + 1),
      }),
    ).toEqual({
      intent: {
        kind: 'sql-prefill',
        sql: 'SELECT 17',
        name: undefined,
      },
    });
  });

  it('does not fall through to SQL when a query ID is oversized', () => {
    expect(
      validateInsightsSearch({
        query_id: 'q'.repeat(MAX_INSIGHTS_QUERY_ID_BYTES + 1),
        sql: "SELECT 'must-not-survive'",
      }),
    ).toEqual({});
  });

  it('discards a name without SQL and unrelated search parameters', () => {
    expect(
      validateInsightsSearch({ name: 'orphan', cel: 'event.data.x' }),
    ).toEqual({});
  });

  it('builds encoded links through the same validation boundary', () => {
    expect(savedQueryInsightsURL('production/eu', '  saved + 17  ')).toBe(
      '/env/production%2Feu/insights?query_id=saved+%2B+17',
    );
    expect(
      sqlPrefillInsightsURL(
        'production/eu',
        "SELECT 'A & B + C' AS marker",
        '  Runs / café  ',
      ),
    ).toBe(
      '/env/production%2Feu/insights?sql=SELECT+%27A+%26+B+%2B+C%27+AS+marker&name=Runs+%2F+caf%C3%A9',
    );
  });

  it('refuses to emit links the route would discard', () => {
    expect(
      savedQueryInsightsURL(
        'production',
        'q'.repeat(MAX_INSIGHTS_QUERY_ID_BYTES + 1),
      ),
    ).toBeUndefined();
    expect(
      sqlPrefillInsightsURL(
        'production',
        'é'.repeat(MAX_INSIGHTS_SQL_BYTES / 2 + 1),
      ),
    ).toBeUndefined();
  });

  it('consumes only one-shot SQL fields after application', () => {
    expect(
      consumeSQLPrefillURL(
        '/env/production/insights?sql=SELECT+17&name=Runs+search&intent=%7B%22kind%22%3A%22sql-prefill%22%7D&keep=asymmetric#result',
      ),
    ).toBe('/env/production/insights?keep=asymmetric#result');
  });

  it('synchronizes saved-query state without retaining command fields', () => {
    expect(
      syncSavedQueryURL(
        '/env/production/insights?query_id=A&sql=SELECT+17&name=stale&intent=%7B%22kind%22%3A%22saved-query%22%7D&keep=29',
        'B',
      ),
    ).toBe('/env/production/insights?keep=29&query_id=B');
    expect(
      syncSavedQueryURL(
        '/env/production/insights?query_id=B&keep=29',
        undefined,
      ),
    ).toBe('/env/production/insights?keep=29');
  });
});
