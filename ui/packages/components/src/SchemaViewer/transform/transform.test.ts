import { describe, expect, it } from 'vitest';

import { TRANSFORM_TEST_CASES } from './tests';
import { transformJSONSchema } from './transform';

describe('transformJSONSchema', () => {
  for (const testCase of TRANSFORM_TEST_CASES) {
    it(testCase.name, () => {
      const tree = transformJSONSchema(testCase.schema);
      expect(tree).toEqual(testCase.expected);
    });
  }
});
