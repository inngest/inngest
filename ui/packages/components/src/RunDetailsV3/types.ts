export type Trace = {
  attempts: number | null;
  childrenSpans?: Trace[];
  endedAt: string | null;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  spanID: string;
  stepID?: string | null;
  startedAt: string | null;
  status: string;
  stepInfo: StepInfoInvoke | StepInfoSleep | StepInfoWait | StepInfoRun | StepInfoSignal | null;
  stepOp?: string | null;
  stepType: string;
  userlandSpan: UserlandSpanType | null;
  isUserland: boolean;
};

export type UserlandSpanType = {
  spanName: string | null;
  spanKind: string | null;
  serviceName: string | null;
  scopeName: string | null;
  scopeVersion: string | null;
  spanAttrs: string | null;
  resourceAttrs: string | null;
};

export type StepInfoInvoke = {
  triggeringEventID: string;
  functionID: string;
  timeout: string;
  returnEventID: string | null;
  runID: string | null;
  timedOut: boolean | null;
};

export type StepInfoSleep = {
  sleepUntil: string;
};

export type StepInfoWait = {
  eventName: string;
  expression: string | null;
  timeout: string;
  foundEventID: string | null;
  timedOut: boolean | null;
};

export type StepInfoRun = {
  type: string | null;
};

export type StepInfoSignal = {
  signal: string;
  timeout: string;
  timedOut: boolean | null;
};

export function isStepInfoRun(stepInfo: Trace['stepInfo']): stepInfo is StepInfoRun {
  if (!stepInfo) {
    return false;
  }

  return 'type' in stepInfo;
}

export function isStepInfoInvoke(stepInfo: Trace['stepInfo']): stepInfo is StepInfoInvoke {
  if (!stepInfo) {
    return false;
  }

  return 'triggeringEventID' in stepInfo;
}

export function isStepInfoSleep(stepInfo: Trace['stepInfo']): stepInfo is StepInfoSleep {
  if (!stepInfo) {
    return false;
  }

  return 'sleepUntil' in stepInfo;
}

export function isStepInfoWait(stepInfo: Trace['stepInfo']): stepInfo is StepInfoWait {
  if (!stepInfo) {
    return false;
  }

  return 'foundEventID' in stepInfo;
}

export function isStepInfoSignal(stepInfo: Trace['stepInfo']): stepInfo is StepInfoSignal {
  if (!stepInfo) {
    return false;
  }

  return 'signal' in stepInfo;
}
