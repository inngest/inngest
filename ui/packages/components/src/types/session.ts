export type SessionFunction = {
  slug: string;
  name: string;
};

export type Session = {
  sessionKey: string;
  sessionId: string;
  runCount: number;
  failedRunCount: number;
  failureRate: number;
  lastActiveAt: string;
  functions: SessionFunction[];
};

export type SessionKey = {
  sessionKey: string;
  createdAt: string;
};

// SessionsPage is one page of a paginated sessions listing; nextCursor is
// null on the last page.
export type SessionsPage = {
  sessions: Session[];
  nextCursor: string | null;
};
