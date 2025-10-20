import type { FunctionDescriptor } from './types';

export const COMPARISON_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'equals', signature: 'equals(${1:a}, ${2:b})' },
  { name: 'greater', signature: 'greater(${1:a}, ${2:b})' },
  { name: 'greaterOrEquals', signature: 'greaterOrEquals(${1:a}, ${2:b})' },
  { name: 'less', signature: 'less(${1:a}, ${2:b})' },
  { name: 'lessOrEquals', signature: 'lessOrEquals(${1:a}, ${2:b})' },
  { name: 'notEquals', signature: 'notEquals(${1:a}, ${2:b})' },
];
