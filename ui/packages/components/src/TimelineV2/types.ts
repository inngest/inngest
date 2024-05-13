export type Trace = {
  attempts: number | null;
  childrenSpans?: Trace[];
  endedAt: string | null;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  spanID: string;
  startedAt: string | null;
  status: string;
  stepInfo?: unknown;
  stepOp?: string | null;
};
