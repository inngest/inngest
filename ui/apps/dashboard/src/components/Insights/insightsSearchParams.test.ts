import { describe, expect, it } from 'vitest';

import {
  MAX_INSIGHTS_NAME_BYTES,
  MAX_INSIGHTS_QUERY_ID_BYTES,
  MAX_INSIGHTS_SQL_BYTES,
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
    ).toEqual({ query_id: 'saved-query-17' });
  });

  it('accepts SQL-prefill mode with an optional name', () => {
    const sql = "SELECT 'space & plus + slash /' AS marker";

    expect(validateInsightsSearch({ sql, name: '  Failed: café  ' })).toEqual({
      sql,
      name: 'Failed: café',
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
    ).toEqual({ sql: 'SELECT 17' });
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
});
