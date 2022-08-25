export type EventTriggers = { [key: string]: any };

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
