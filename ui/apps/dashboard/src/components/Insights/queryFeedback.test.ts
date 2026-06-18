import { describe, expect, it } from 'vitest';

import { sqlWasEdited } from './queryFeedback';

describe('sqlWasEdited', () => {
  it('is false when only formatting, whitespace, or case differ', () => {
    expect(
      sqlWasEdited('select count() from runs', 'SELECT count()\nFROM   runs'),
    ).toBe(false);
  });

  it('is false for an exact match', () => {
    expect(sqlWasEdited('SELECT 1', 'SELECT 1')).toBe(false);
  });

  it('tolerates a trailing-semicolon difference', () => {
    expect(sqlWasEdited('SELECT 1', 'SELECT 1;')).toBe(false);
  });

  it('is true when the query was materially changed', () => {
    expect(sqlWasEdited('SELECT 1', 'SELECT 2')).toBe(true);
  });

  it('is true when there is no suggestion to compare against', () => {
    expect(sqlWasEdited(undefined, 'SELECT 1')).toBe(true);
  });
});
