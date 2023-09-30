import { IconEvent, IconExclamationTriangle } from '@/icons';
import { FunctionRunStatus, type FunctionRun } from '../../store/generated';
import classNames from '../../utils/classnames';
import renderFuncCardFooter from './FuncCardFooterRenderer';
import { FunctionRunExtraStatus } from './RunStatus';

interface FuncCardFooterProps {
  functionRun: Omit<FunctionRun, 'history' | 'functionID' | 'historyItemOutput'>;
}

type StatusConfig = {
  component: React.ComponentType<{}>;
  color: string;
};

export default function FuncCardFooter({ functionRun }: FuncCardFooterProps) {
  const { eventName, stepName, message, errorName, status, time } =
    renderFuncCardFooter(functionRun);
  const functionRunStatusFooter = {
    [FunctionRunStatus.Running]: {
      component: () => {
        if (!stepName) return null;
        return (
          <p>
            Step <span className="font-bold truncate">{stepName}</span> is running
          </p>
        );
      },
      color: 'bg-sky-500/10',
    },
    [FunctionRunStatus.Completed]: {
      component: () => {
        if (!message) return null;
        return <p className="font-mono text-teal-400 truncate">{message}</p>;
      },
      color: 'bg-teal-500/10',
    },
    [FunctionRunStatus.Failed]: {
      component: () => {
        if (!message && !errorName) return null;
        return (
          <p className="font-mono flex items-center gap-1">
            <IconExclamationTriangle className="icon-2xs text-rose-400" />
            <span className="text-rose-400 font-semibold">{errorName}</span>
            <span className="truncate">{message}</span>
          </p>
        );
      },
      color: 'bg-rose-600/10',
    },
    [FunctionRunStatus.Cancelled]: {
      component: () => {
        if (!eventName) return null;
        return (
          <p className="flex items-center gap-3">
            Cancelled by:{' '}
            <span className="flex gap-1 items-center font-medium	text-slate-300 text-xs truncate">
              <IconEvent className="icon-2xs" />
              {eventName}
            </span>
          </p>
        );
      },
      color: 'bg-slate-500/10',
    },
    [FunctionRunExtraStatus.WaitingFor]: {
      component: () => {
        if (!eventName) return null;
        return (
          <p className="flex items-center gap-3">
            Waiting for:{' '}
            <span className="flex gap-1 items-center font-medium	text-slate-300 text-xs truncate">
              <IconEvent className="icon-2xs" />
              {eventName}
            </span>
          </p>
        );
      },
      color: 'bg-sky-500/10',
    },
    [FunctionRunExtraStatus.Sleeping]: {
      component: () => {
        if (!time) return null;
        return <p>Sleeping until: {time}</p>;
      },
      color: 'bg-sky-500/10',
    },
  } as const satisfies Record<FunctionRunStatus | FunctionRunExtraStatus, StatusConfig>;

  const content = functionRunStatusFooter[status].component();
  const backgroundColor = functionRunStatusFooter[status].color;

  if (!content) return null;

  return (
    <div className={classNames(backgroundColor, 'text-2xs px-5 py-4 text-slate-100')}>
      {content}
    </div>
  );
}
