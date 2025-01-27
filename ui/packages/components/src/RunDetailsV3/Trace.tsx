import { useState, type ReactNode } from 'react';

import { Timeline } from './Timeline';
import { Workflow } from './Workflow';

export const TraceTab = ({
  active,
  children,
  onClick,
}: {
  active: boolean;
  children: ReactNode;
  onClick: () => void;
}) => {
  return (
    <div
      className={`${
        active ? 'text-basis border-contrast ' : 'text-muted border-transparent'
      }  flex h-[30px] cursor-pointer items-center self-center border-b-2 text-sm leading-tight outline-none`}
      onClick={onClick}
    >
      {children}
    </div>
  );
};

export const Trace = () => {
  const [active, setActive] = useState<'trace' | 'workflow'>('trace');

  return (
    <div className="flex flex-col">
      <div className="border-muted flex w-fit flex-row gap-4 border-b">
        <TraceTab active={active === 'trace'} onClick={() => setActive('trace')}>
          Trace View
        </TraceTab>
        <TraceTab active={active === 'workflow'} onClick={() => setActive('workflow')}>
          Workflow View
        </TraceTab>
      </div>
      <div>{active === 'trace' ? <Timeline /> : <Workflow />}</div>
    </div>
  );
};
