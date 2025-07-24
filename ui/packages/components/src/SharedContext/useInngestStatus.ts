import { z } from 'zod';

import { useShared } from './SharedContext';

export const impactSchema = z.enum(['partial_outage', 'degraded_performance', 'full_outage']);
export const indicatorSchema = z.enum(['none', 'maintenance', ...impactSchema.options]);
export type Indicator = z.infer<typeof indicatorSchema>;

export type InngestStatus = {
  url: string;
  description: string;
  impact: Indicator;
  indicatorColor: string;
  updated_at: string;
};

export const useRerun = () => {
  const shared = useShared();
  const inngestStatus = (): InngestStatus | null => shared.inngestStatus;

  return {
    inngestStatus,
  };
};
