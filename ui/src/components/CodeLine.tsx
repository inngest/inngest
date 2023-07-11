import { useState } from 'react';
import { IconCopy, IconStatusCompleted } from '@/icons';
import classNames from '@/utils/classnames';

type CodeLineProps = {
  code: string;
  className?: string;
};

export default function CodeLine({ code, className }: CodeLineProps) {
  const [clickedState, setClickedState] = useState(false);
  const handleCopyClick = (code) => {
    setClickedState(true);
    navigator.clipboard.writeText(code);
    setTimeout(() => {
      setClickedState(false);
    }, 1000);
  };
  return (
    <div
      className={classNames(
        className,
        'flex items-center justify-between bg-slate-950 rounded-md cursor-pointer'
      )}
      onClick={() => handleCopyClick(code)}
    >
      <code className="text-slate-300">{code}</code>
      {clickedState ? <IconStatusCompleted /> : <IconCopy />}
    </div>
  );
}
