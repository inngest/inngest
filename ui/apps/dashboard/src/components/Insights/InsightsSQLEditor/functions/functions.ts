import { AGGREGATE_FUNCTIONS } from './aggregate';
import { ARITHMETIC_FUNCTIONS } from './arithmetic';
import { COMPARISON_FUNCTIONS } from './comparison';
import { JSON_FUNCTIONS } from './json';
import { NULLABLE_FUNCTIONS } from './nullable';
import { ROUNDING_FUNCTIONS } from './rounding';
import type { FunctionDescriptor } from './types';
import { ULID_FUNCTIONS } from './ulid';

export const SUPPORTED_FUNCTIONS = sortByName([
  ...AGGREGATE_FUNCTIONS,
  ...ARITHMETIC_FUNCTIONS,
  ...COMPARISON_FUNCTIONS,
  ...JSON_FUNCTIONS,
  ...NULLABLE_FUNCTIONS,
  ...ROUNDING_FUNCTIONS,
  ...ULID_FUNCTIONS,
]);

function sortByName(fns: FunctionDescriptor[]): FunctionDescriptor[] {
  return fns.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
}
