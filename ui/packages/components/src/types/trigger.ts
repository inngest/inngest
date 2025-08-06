export type Trigger = {
  type: TriggerTypes;
  value: string;
};

export enum TriggerTypes {
  Event = 'EVENT',
  Cron = 'CRON',
}
