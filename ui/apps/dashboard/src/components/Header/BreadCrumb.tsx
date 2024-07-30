import { RiArrowRightSLine } from '@remixicon/react';

export const BreadCrumb = ({ path }: { path: string[] }) => {
  return path.map((part: string, i: number) => {
    const last = i === path.length - 1;
    return (
      <div className="flex flex-row items-center justify-start" key={`${path}-key-${i}`}>
        <span className={`${last ? 'text-basis' : 'text-subtle'} mr-2 text-sm leading-tight`}>
          {path}
        </span>
        {!last && <RiArrowRightSLine className="text-muted mr-2 h-5 w-5" />}
      </div>
    );
  });
};
