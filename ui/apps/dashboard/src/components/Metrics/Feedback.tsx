import { Button } from '@inngest/components/Button';

import { ChartIcon } from '@/icons/ChartIcon';

export const Feedback = ({}) => {
  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col items-center justify-center gap-3 overflow-x-hidden rounded-md border p-2 text-center md:w-[65%] md:px-12 ">
      <ChartIcon />
      <div className="text-lg font-medium">Can&apos;t find the data you need?</div>
      <div className="text-subtle text-sm leading-tight">
        Let our team know which charts are most useful to you and request any additional charts that
        you might need but are currently missing.
      </div>
      <div className="flex flex-row items-center justify-center gap-2">
        <Button
          kind="secondary"
          appearance="outlined"
          label="Request charts"
          href="https://roadmap.inngest.com/roadmap"
          target="_new"
        />
      </div>
    </div>
  );
};
