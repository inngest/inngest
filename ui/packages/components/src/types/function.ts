export type Function = {
  id: string;
  name: string;
  triggers: {
    type: 'CRON' | 'EVENT';
    value: string;
  }[];
};
