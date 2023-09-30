import { FunctionRunStatus } from '@/store/generated';

export enum FunctionRunExtraStatus {
  WaitingFor = 'WAITINGFOR',
  Sleeping = 'SLEEPING',
}

export function renderRunStatus(functionRun) {
  if (!functionRun) return null;
  const isWaitingFor = functionRun.waitingFor?.eventName;
  const isSleeping = functionRun.waitingFor?.expiryTime;

  if (isWaitingFor) {
    return FunctionRunExtraStatus.WaitingFor;
  } else if (isSleeping) {
    return FunctionRunExtraStatus.Sleeping;
  } else {
    return functionRun.status || FunctionRunStatus.Completed;
  }
}
