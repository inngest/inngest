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
