export type EventTriggers = ApiUserCreated;

export interface ApiUserCreated {
  name: string;
  data: {
    email: string;
    plan: "free" | "starter" | "pro";
  };
  user: {
    email: string;
    external_id: string;
  };
  ts: number;
};

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
