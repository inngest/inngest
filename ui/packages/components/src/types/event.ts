export type Event = {
  id: string;
  name: string;
  payload: string;
  receivedAt: Date;
  source?: string;
  version?: string;
  idempotencyKey?: string;
  timestamp?: Date;
  runs?: {
    fnName?: string;
    fnSlug?: string;
    id: string;
    status: string;
    startedAt?: Date;
    completedAt?: Date;
  }[];
};

export type PageInfo = {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
};
