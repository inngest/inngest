import { Shape } from "src/types";

export type Event = {
  id: string;
  name: string;
  version: string;
  fields: Shape;
  createdAt: string;
};

export type RecentEvent = {
  id: string;
  receivedAt: string;
  occurredAt: string;
  name: string;
  event: string;
  version: string;
  contactID: string;
  contact?: {
    id: string;
    predefinedAttributes: {
      first_name?: string;
      last_name?: string;
      email?: string;
    };
    attributes: Array<{
      name: string;
      value: string;
    }>;
  };
};
