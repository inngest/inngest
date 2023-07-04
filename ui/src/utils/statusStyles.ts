import {
  IconStatusActionReq,
  IconStatusCompleted,
  IconStatusDefault,
  IconStatusFailed,
  IconStatusNoFn,
  IconStatusPaused,
  IconStatusRunning,
} from '../icons';
import { EventStatus, FunctionRunStatus } from '../store/generated';

export enum FunctionStatus {
  Registered = 'REGISTERED',
}

export default function statusStyles(
  status: EventStatus | FunctionRunStatus | FunctionStatus | null
) {
  switch (status) {
    case FunctionRunStatus.Running:
    case EventStatus.Running:
      return {
        text: 'text-white',
        icon: IconStatusRunning,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionRunStatus.Completed:
    case EventStatus.Completed:
      return {
        text: 'text-white',
        icon: IconStatusCompleted,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionRunStatus.Failed:
    case EventStatus.Failed:
      return {
        text: 'text-red-400',
        icon: IconStatusFailed,
        fnBG: 'bg-red-400/20 group-hover:bg-red-400/40',
      };
    case EventStatus.Paused:
      return {
        text: 'text-white',
        icon: IconStatusPaused,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case EventStatus.PartiallyFailed:
      return {
        text: 'text-orange-300',
        icon: IconStatusActionReq,
        fnBG: 'bg-yellow-500/20 group-hover:bg-yellow-500/40',
      };
    case EventStatus.NoFunctions:
      return {
        text: 'text-white',
        icon: IconStatusNoFn,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    case FunctionStatus.Registered:
      return {
        text: 'text-white',
        icon: IconStatusCompleted,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
    default:
      return {
        text: 'text-white',
        icon: IconStatusDefault,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      };
  }
}
