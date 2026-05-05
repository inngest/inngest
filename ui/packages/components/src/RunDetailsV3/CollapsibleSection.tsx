import { useState } from 'react';
import { RiArrowRightSLine } from '@remixicon/react';

type Props = {
  title: string;
  children: React.ReactNode;
};

export const CollapsibleSection = ({ title, children }: Props) => {
  const [expanded, setExpanded] = useState(true);

  return (
    <div className="mb-4 flex flex-col gap-2">
      <div
        className="text-basis flex h-11 cursor-pointer items-center gap-2"
        onClick={() => setExpanded(!expanded)}
      >
        <RiArrowRightSLine
          className={`shrink-0 transition-transform duration-[250ms] ${
            expanded ? 'rotate-90' : ''
          }`}
        />
        <span className="text-basis text-sm font-normal">{title}</span>
      </div>
      {expanded && children}
    </div>
  );
};
