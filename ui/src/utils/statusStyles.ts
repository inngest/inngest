import {
  DeprecatedIconStatusActionReq,
  DeprecatedIconStatusCompleted,
  DeprecatedIconStatusDefault,
  DeprecatedIconStatusFailed,
  DeprecatedIconStatusNoFn,
  DeprecatedIconStatusPaused,
  DeprecatedIconStatusRunning,
} from '../icons';
import { EventStatus, FunctionRunStatus } from '../store/generated';

export enum FunctionStatus {
  Registered = 'REGISTERED',
}

export default function statusStyles(
  status: EventStatus | FunctionRunStatus | FunctionStatus | null,
) {
  switch (status) {
    case FunctionRunStatus.Running:
    case EventStatus.Running:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusRunning,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionRunStatus.Completed:
    case EventStatus.Completed:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusCompleted,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionRunStatus.Failed:
    case EventStatus.Failed:
      return {
        text: 'text-red-400',
        icon: DeprecatedIconStatusFailed,
        fnBG: 'bg-red-400/20 group-hover:bg-red-400/40',
      };
    case EventStatus.Paused:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusPaused,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case EventStatus.PartiallyFailed:
      return {
        text: 'text-orange-300',
        icon: DeprecatedIconStatusActionReq,
        fnBG: 'bg-yellow-500/20 group-hover:bg-yellow-500/40',
      };
    case EventStatus.NoFunctions:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusNoFn,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionStatus.Registered:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusCompleted,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    default:
      return {
        text: 'text-white',
        icon: DeprecatedIconStatusDefault,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
  }
}
