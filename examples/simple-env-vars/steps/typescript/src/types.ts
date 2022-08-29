// Generated via inngest init

export interface DemoEventSent {
  data: {
    message: string;
  };
  ts: number;
  name: string;
};

export type EventTriggers = DemoEventSent;

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
