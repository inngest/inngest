export type Event = {
  id: string;
  name: string;
  payload?: string;
  receivedAt: string;
  functions?: {
    name: string;
    slug: string;
    status: string;
  }[];
};

export type PageInfo = {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
};
