import { AGGREGATE_FUNCTIONS } from './aggregate';
import { ARITHMETIC_FUNCTIONS } from './arithmetic';
import { COMPARISON_FUNCTIONS } from './comparison';
import { ROUNDING_FUNCTIONS } from './rounding';
import type { FunctionDescriptor } from './types';

export const SUPPORTED_FUNCTIONS = sortByName([
  ...AGGREGATE_FUNCTIONS,
  ...ARITHMETIC_FUNCTIONS,
  ...COMPARISON_FUNCTIONS,
  ...ROUNDING_FUNCTIONS,
]);

function sortByName(fns: FunctionDescriptor[]): FunctionDescriptor[] {
  return fns.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
}
