import { useEffect, useState } from 'react';
import { parseDuration, subtractDuration } from '@inngest/components/utils/date';

type CalculatedStartTimeProps = {
  lastDays?: string;
  startTime?: string;
};

export const DEFAULT_TIME = '3d';

export const useCalculatedStartTime = ({ lastDays, startTime }: CalculatedStartTimeProps): Date => {
  const [calculatedStartTime, setCalculatedStartTime] = useState<Date>(new Date());

  useEffect(() => {
    if (lastDays) {
      setCalculatedStartTime(subtractDuration(new Date(), parseDuration(lastDays)));
    } else if (startTime) {
      setCalculatedStartTime(new Date(startTime));
    } else {
      setCalculatedStartTime(subtractDuration(new Date(), parseDuration(DEFAULT_TIME)));
    }
  }, [lastDays, startTime]);

  return calculatedStartTime;
};
