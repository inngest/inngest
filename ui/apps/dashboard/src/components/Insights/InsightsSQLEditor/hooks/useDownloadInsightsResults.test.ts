import { describe, expect, it } from 'vitest';

import type { InsightsFetchResult } from '../../InsightsStateMachineContext/types';
import { convertToCSV, convertToJSON } from './useDownloadInsightsResults';

const data: InsightsFetchResult = {
  columns: [{ name: 'created_at', type: 'date' }],
  rows: [
    {
      id: 'row-0',
      values: { created_at: new Date('invalid') },
    },
  ],
  diagnostics: [],
};

describe('convertToCSV', () => {
  it('does not throw when a date value is invalid', () => {
    expect(convertToCSV(data)).toBe('created_at\n');
  });
});

describe('convertToJSON', () => {
  it('serializes an invalid date as null', () => {
    expect(convertToJSON(data)).toBe('[\n  {\n    "created_at": null\n  }\n]');
  });
});
