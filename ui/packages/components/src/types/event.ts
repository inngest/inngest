export type Event = {
  id: string;
  name: string;
  payload?: string;
  receivedAt: Date;
  source?: {
    name?: string | null;
  } | null;
  version?: string | null;
  idempotencyKey?: string | null;
  occurredAt?: Date;
  runs?: {
    fnName?: string;
    fnSlug?: string;
    id: string;
    status: string;
    startedAt?: Date;
    completedAt?: Date;
    skipReason?: string;
    skipExistingRunID?: string;
  }[];
};

export type PageInfo = {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
};
