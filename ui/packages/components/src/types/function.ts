export type Function = {
  id: string;
  name: string;
  slug: string;
  triggers: {
    type: 'CRON' | 'EVENT';
    value: string;
  }[];
  version?: number | null;
};
