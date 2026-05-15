import type { FunctionDescriptor } from './types';

export const ULID_FUNCTIONS: FunctionDescriptor[] = [
  {
    name: 'ULIDStringToDateTime',
    signature: 'ULIDStringToDateTime(${1:ulid})',
  },
];
