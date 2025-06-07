import type { BooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag.js';

export const useBooleanFlag = (flag: string, defaultValue: boolean = false): BooleanFlag => {
  if (flag === 'step-over-debugger') {
    return { isReady: true, value: true };
  }

  return { isReady: true, value: defaultValue };
};
