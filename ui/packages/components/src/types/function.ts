export type Function = {
  name: string;
  triggers: {
    type: 'CRON' | 'EVENT';
    value: string;
  }[];
};
