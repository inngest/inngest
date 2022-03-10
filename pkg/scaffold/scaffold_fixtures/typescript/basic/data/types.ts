{{ EventTypes "typescript" }}

export type Args = {
  event: EventTriggers;
  actions: {
    [clientID: string]: any
  };
};

