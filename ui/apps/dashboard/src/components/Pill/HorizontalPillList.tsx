import * as Tooltip from '@radix-ui/react-tooltip';

import { Pill } from '@/components/Pill/Pill';

type FunctionsCellContentProps = {
  pills: React.ReactNode[];
  alwaysVisibleCount?: number;
};

export default function HorizontalPillList({
  pills,
  alwaysVisibleCount,
}: FunctionsCellContentProps) {
  if (pills.length === 0) return null;

  if (alwaysVisibleCount && pills.length > alwaysVisibleCount) {
    const hiddenPills = pills.slice(alwaysVisibleCount);
    const alwaysVisiblePills = pills.slice(0, alwaysVisibleCount);

    return (
      <>
        {alwaysVisiblePills}
        <Tooltip.Provider>
          <Tooltip.Root delayDuration={0}>
            <Tooltip.Trigger className="cursor-default">
              <Pill className="bg-white px-2.5 align-middle text-slate-600">
                +{hiddenPills.length}
              </Pill>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                className="data-[state=delayed-open]:data-[side=top]:animate-slideDownAndFade data-[state=delayed-open]:data-[side=right]:animate-slideLeftAndFade data-[state=delayed-open]:data-[side=left]:animate-slideRightAndFade data-[state=delayed-open]:data-[side=bottom]:animate-slideUpAndFade text-violet11 select-none rounded-[4px] bg-white px-[15px] py-[10px] text-[15px] leading-none shadow-[hsl(206_22%_7%_/_35%)_0px_10px_38px_-10px,_hsl(206_22%_7%_/_20%)_0px_10px_20px_-15px] will-change-[transform,opacity]"
                sideOffset={5}
              >
                <div className="flex flex-col gap-2">{hiddenPills}</div>
                <Tooltip.Arrow className="fill-white" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>
        </Tooltip.Provider>
      </>
    );
  }

  return <>{pills}</>;
}
