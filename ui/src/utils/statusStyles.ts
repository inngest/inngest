import {
  IconStatusRunning,
  IconStatusCompleted,
  IconStatusFailed,
  IconStatusPaused,
  IconStatusActionReq,
  IconStatusNoFn,
  IconStatusDefault,
} from '../icons'


export default function getStatusVals(status) {
  switch (status) {
    case 'RUNNING':
      return {
        text: 'text-white',
        icon: IconStatusRunning,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      }
    case 'COMPLETED':
      return {
        text: 'text-white',
        icon: IconStatusCompleted,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      }
    case 'FAILED':
      return {
        text: 'text-red-400',
        icon: IconStatusFailed,
        fnBG: 'bg-red-400/20 group-hover:bg-red-400/40',
      }
    case 'PAUSED':
      return {
        text: 'text-white',
        icon: IconStatusPaused,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      }
    case 'ACTION_REQ':
      return {
        text: 'text-orange-300',
        icon: IconStatusActionReq,
        fnBG: 'bg-yellow-500/20 group-hover:bg-yellow-500/40',
      }
    case 'NO_FN':
      return {
        text: 'text-white',
        icon: IconStatusNoFn,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      }
    default:
      return {
        text: 'text-white',
        icon: IconStatusDefault,
        fnBG: 'bg-slate-800 group-hover:bg-slate-700',
      }
  }
}