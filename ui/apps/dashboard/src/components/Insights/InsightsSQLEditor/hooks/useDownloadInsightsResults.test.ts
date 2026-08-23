import { describe, expect, it } from 'vitest';

import type { InsightsFetchResult } from '../../InsightsStateMachineContext/types';
import { convertToCSV } from './useDownloadInsightsResults';

describe('convertToCSV', () => {
  it('does not throw when a date value is invalid', () => {
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

    expect(convertToCSV(data)).toBe('created_at\n');
  });
});
