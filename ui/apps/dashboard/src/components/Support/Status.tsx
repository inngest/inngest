import { z } from 'zod';

const impactSchema = z.enum(['partial_outage', 'degraded_performance', 'full_outage']);

const indicatorSchema = z.enum(['none', 'maintenance', ...impactSchema.options]);
type Indicator = z.infer<typeof indicatorSchema>;

const statusEventSchema = z.object({
  id: z.string(),
  name: z.string(),
  url: z.string(),
  last_update_at: z.string(),
  last_update_message: z.string(),
  affected_components: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      group_name: z.string().optional(),
    })
  ),
});

const incidentSchema = statusEventSchema.extend({
  status: z.enum(['identified', 'investigating', 'monitoring']),
  current_worst_impact: impactSchema,
});

const maintenanceInProgressEventSchema = statusEventSchema.extend({
  status: z.enum(['maintenance_in_progress']),
  started_at: z.string(),
  scheduled_end_at: z.string(),
});

const maintenanceScheduledEventSchema = statusEventSchema.extend({
  status: z.enum(['maintenance_scheduled']),
  starts_at: z.string(),
  ends_at: z.string(),
});

const statusPageSummaryResponseSchema = z.object({
  page_title: z.string(),
  page_url: z.string(),
  ongoing_incidents: z.array(incidentSchema),
  in_progress_maintenances: z.array(maintenanceInProgressEventSchema),
  scheduled_maintenances: z.array(maintenanceScheduledEventSchema),
});

export type Status = {
  url: string;
  description: string;
  impact: Indicator;
  indicatorColor: string;
  updated_at: string;
};

const impactMessage: { [K in Indicator]: string } = {
  none: 'All systems operational',
  degraded_performance: 'Degraded performance',
  partial_outage: 'Partial system outage',
  full_outage: 'Major system outage',
  maintenance: 'Maintenance in progress',
};

export const indicatorColor: { [K in Indicator]: string } = {
  none: 'rgba(var(--color-matcha-500))',
  degraded_performance: 'rgb(var(--color-honey-300))',
  maintenance: 'rgb(var(--color-honey-300))',
  partial_outage: 'rgb(var(--color-honey-500))',
  full_outage: 'rgb(var(--color-ruby-500))',
};

export const STATUS_PAGE_URL = 'https://status.inngest.com';

const mapStatus = (res: z.infer<typeof statusPageSummaryResponseSchema>): Status => {
  // Grab first incident and maintenance item
  const incident = res.ongoing_incidents[0];
  const maintenance = res.in_progress_maintenances[0];
  const impact: Indicator =
    incident?.current_worst_impact || (maintenance ? 'maintenance' : 'none');
  return {
    indicatorColor: indicatorColor[impact],
    impact,
    description: impactMessage[impact],
    updated_at: incident?.last_update_at || new Date().toString(),
    url: incident?.url || STATUS_PAGE_URL,
  };
};

const fetchStatus = async () => {
  return statusPageSummaryResponseSchema.parse(
    await fetch('https://status.inngest.com/api/v1/summary').then((r) => r.json())
  );
};

export const getStatus = async (): Promise<Status | undefined> => {
  try {
    return mapStatus(await fetchStatus());
  } catch (e) {
    console.error(e);
    return undefined;
  }
};
