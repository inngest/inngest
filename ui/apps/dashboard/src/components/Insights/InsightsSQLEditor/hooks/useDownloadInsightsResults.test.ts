import { describe, expect, it } from 'vitest';

import type { InsightsFetchResult } from '../../InsightsStateMachineContext/types';
import { InsightsColumnRole } from '@/gql/graphql';
import { convertToCSV, convertToJSON } from './useDownloadInsightsResults';

const data: InsightsFetchResult = {
  columns: [
    {
      name: 'created_at',
      type: 'date',
      role: InsightsColumnRole.Unspecified,
    },
  ],
  rows: [
    {
      id: 'row-0',
      values: { created_at: new Date('invalid') },
    },
  ],
  diagnostics: [],
};

function resultOf(
  names: string[],
  rows: Array<Array<string | number | null>>,
): InsightsFetchResult {
  return {
    columns: names.map((name) => ({
      name,
      type: 'string',
      role: InsightsColumnRole.Unspecified,
    })),
    rows: rows.map((values, index) => ({
      id: `row-${index}`,
      values: Object.fromEntries(names.map((name, i) => [name, values[i]])),
    })),
    diagnostics: [],
  };
}

describe('convertToCSV', () => {
  it('does not throw when a date value is invalid', () => {
    expect(convertToCSV(data)).toBe('created_at\n');
  });

  it('leaves plain headers and values unquoted', () => {
    expect(
      convertToCSV(
        resultOf(['event_name', 'total'], [['app/user.created', 3]]),
      ),
    ).toBe('event_name,total\napp/user.created,3');
  });

  it('quotes values that contain a comma or a double quote', () => {
    expect(convertToCSV(resultOf(['a', 'b'], [['x, y', 'say "hi"']]))).toBe(
      'a,b\n"x, y","say ""hi"""',
    );
  });

  it('quotes a header that contains a comma', () => {
    // e.g. SELECT if(status = 200, 1, 0), count(*)
    expect(
      convertToCSV(resultOf(['if(status = 200, 1, 0)', 'count(*)'], [[1, 42]])),
    ).toBe('"if(status = 200, 1, 0)",count(*)\n1,42');
  });

  it('escapes double quotes in a header', () => {
    expect(convertToCSV(resultOf(['the "name"', 'n'], [['a', 1]]))).toBe(
      '"the ""name""",n\na,1',
    );
  });

  it('quotes a header that contains a line break or outer spaces', () => {
    expect(
      convertToCSV(resultOf(['line\nbreak', ' padded '], [['a', 'b']])),
    ).toBe('"line\nbreak"," padded "\na,b');
  });

  it('quotes a header the same way as a value with the same text', () => {
    expect(
      convertToCSV(
        resultOf(['first, second', 'third'], [['first, second', 'third']]),
      ),
    ).toBe('"first, second",third\n"first, second",third');
  });
});

describe('convertToJSON', () => {
  it('serializes an invalid date as null', () => {
    expect(convertToJSON(data)).toBe('[\n  {\n    "created_at": null\n  }\n]');
  });
});
