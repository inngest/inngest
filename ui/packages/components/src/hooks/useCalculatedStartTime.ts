import { useMemo } from 'react';
import { parseDuration, subtractDuration } from '@inngest/components/utils/date';

type CalculatedStartTimeProps = {
  lastDays?: string;
  startTime?: string;
  defaultTime?: string;
};

export const DEFAULT_TIME = '3d';

export const useCalculatedStartTime = ({
  lastDays,
  startTime,
  defaultTime,
}: CalculatedStartTimeProps): Date => {
  return useMemo(() => {
    if (lastDays) {
      return subtractDuration(new Date(), parseDuration(lastDays));
    } else if (startTime) {
      return new Date(startTime);
    } else {
      return subtractDuration(new Date(), parseDuration(defaultTime ?? DEFAULT_TIME));
    }
  }, [lastDays, startTime, defaultTime]);
};
