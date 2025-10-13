import type { FunctionDescriptor } from './types';

export const ROUNDING_FUNCTIONS: FunctionDescriptor[] = [
  { name: 'ceiling', signature: 'ceiling(${1:val})' },
  { name: 'floor', signature: 'floor(${1:val})' },
  { name: 'round', signature: 'round(${1:val}, ${2:N})' },
  { name: 'roundAge', signature: 'roundAge(${1:val})' },
  { name: 'roundBankers', signature: 'roundBankers(${1:val}, ${2:N})' },
  { name: 'roundDuration', signature: 'roundDuration(${1:val})' },
  { name: 'roundToExp2', signature: 'roundToExp2(${1:val})' },
  { name: 'truncate', signature: 'truncate(${1:val}, ${2:N})' },
];
