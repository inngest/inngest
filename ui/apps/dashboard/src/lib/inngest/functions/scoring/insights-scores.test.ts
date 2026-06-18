import { describe, expect, it } from 'vitest';

import {
  buildImmediateScores,
  isOpusVariant,
  isParseableSelect,
  producedSql,
} from './insights-scores';

describe('producedSql', () => {
  it('is true when SQL is non-empty', () => {
    expect(producedSql('SELECT 1')).toBe(true);
  });

  it('is false for empty, whitespace, or missing SQL', () => {
    expect(producedSql('')).toBe(false);
    expect(producedSql('   ')).toBe(false);
    expect(producedSql(undefined)).toBe(false);
  });
});

describe('isParseableSelect', () => {
  it('accepts a single SELECT (case-insensitive)', () => {
    expect(isParseableSelect('SELECT 1')).toBe(true);
    expect(isParseableSelect('select count() from runs')).toBe(true);
  });

  it('accepts a WITH … SELECT CTE', () => {
    expect(isParseableSelect('WITH x AS (SELECT 1) SELECT * FROM x')).toBe(
      true,
    );
  });

  it('tolerates a trailing semicolon and surrounding whitespace', () => {
    expect(isParseableSelect('  SELECT 1;  ')).toBe(true);
  });

  it('rejects multiple statements', () => {
    expect(isParseableSelect('SELECT 1; SELECT 2')).toBe(false);
  });

  it('rejects non-SELECT statements', () => {
    expect(isParseableSelect('DROP TABLE runs')).toBe(false);
  });

  it('rejects empty or missing SQL', () => {
    expect(isParseableSelect('')).toBe(false);
    expect(isParseableSelect(undefined)).toBe(false);
  });
});

describe('isOpusVariant', () => {
  it('is true only for the opus variant', () => {
    expect(isOpusVariant('claude-opus-4-8')).toBe(true);
    expect(isOpusVariant('claude-sonnet-4-5')).toBe(false);
  });
});

describe('buildImmediateScores', () => {
  it('emits produced_sql, sql_parseable, and isOpus for a valid opus run', () => {
    expect(
      buildImmediateScores({ sql: 'SELECT 1', variant: 'claude-opus-4-8' }),
    ).toEqual([
      { name: 'insights_produced_sql', value: true },
      { name: 'insights_sql_parseable', value: true },
      { name: 'isOpus', value: true },
    ]);
  });

  it('reflects a clarification turn (no SQL) on a sonnet run', () => {
    expect(
      buildImmediateScores({ sql: '', variant: 'claude-sonnet-4-5' }),
    ).toEqual([
      { name: 'insights_produced_sql', value: false },
      { name: 'insights_sql_parseable', value: false },
      { name: 'isOpus', value: false },
    ]);
  });
});
