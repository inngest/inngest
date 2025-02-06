import { useState, type ReactNode } from 'react';

import { Input } from './Input';
import { Output } from './Output';

export const IOTab = ({
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

export const IO = () => {
  const [active, setActive] = useState<'input' | 'output'>('output');

  return (
    <div className="flex h-full flex-col">
      <div className="border-muted flex h-full w-fit flex-row gap-4 border-b">
        <IOTab active={active === 'input'} onClick={() => setActive('input')}>
          Input
        </IOTab>
        <IOTab active={active === 'output'} onClick={() => setActive('output')}>
          Output
        </IOTab>
      </div>
      <div>{active === 'input' ? <Input /> : <Output />}</div>
    </div>
  );
};
