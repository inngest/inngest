// Generated via inngest init

export interface AuthAccountCreated {
  name: "auth/account.created";
  data: {
    account_id: string;
    method: string;
    plan_name: string;
    subscribed?: boolean;
  };
  user: {
    email: string;
    external_id: string;
    plan_name: string;
  };
  v: "1";
  ts: number;
};

export type EventTriggers = AuthAccountCreated;

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
