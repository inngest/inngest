import type { FunctionDescriptor } from './types';

export const NULLABLE_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'assumeNotNull', signature: 'assumeNotNull(${1:x})' },
  { name: 'coalesce', signature: 'coalesce(${1:x}, ${2:y})' },
  { name: 'ifNull', signature: 'ifNull(${1:x}, ${2:y})' },
  // { name: 'isNotDistinctFrom', signature: 'isNotDistinctFrom(${1:x}, ${2:y})' }, error
  { name: 'isNotNull', signature: 'isNotNull(${1:x})' },
  { name: 'isNullable', signature: 'isNullable(${1:x})' },
  { name: 'isZeroOrNull', signature: 'isZeroOrNull(${1:x})' },
  { name: 'isNull', signature: 'isNull(${1:x})' },
  // { name: 'nullIf', signature: 'nullIf(${1:x}, ${2:y})' }, base 16 (<nil> if both null)
  // { name: 'toNullable', signature: 'toNullable(${1:x})' }, base 16 (<nil> if both null)
];
