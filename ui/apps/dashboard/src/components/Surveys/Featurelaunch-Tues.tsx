'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button/index';
import { RiCloseLine } from '@remixicon/react';

const HIDE_FEATURE_LAUNCH_TUES = 'inngest-feature-launch-tues-hide';

export default function FeaturelaunchTues() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Check if already dismissed
    if (window.localStorage.getItem(HIDE_FEATURE_LAUNCH_TUES) === 'true') return;

    // Show the component (no date restrictions)
    setOpen(true);
  }, []);

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_FEATURE_LAUNCH_TUES, 'true');
  };

  return (
    open && (
      <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[430px] rounded border">
        <div className="gap-x border-subtle flex flex-row items-center justify-between border-b p-3">
          <div className="text-sm font-medium leading-tight">Launch week day 2: Insights</div>
          <Button
            icon={<RiCloseLine className="text-subtle h-5 w-5" />}
            kind="secondary"
            appearance="ghost"
            size="small"
            className="ml-.5"
            onClick={() => dismiss()}
          />
        </div>
        <div className="text-muted px-3 pb-3 pt-3 text-sm">
          You can now query events to analyze trends, usage, and performance right where your
          workflows run. No tool switching required
          <div className="flex gap-2 pt-2">
            <button
              className="bg-btnPrimary focus:bg-btnPrimaryHover hover:bg-btnPrimaryHover active:bg-btnPrimaryActive disabled:bg-btnPrimaryDisabled disabled:text-btnPrimaryDisabled relative flex h-8 items-center justify-center justify-items-center whitespace-nowrap rounded-md px-3 py-1.5 text-xs leading-[18px] text-white disabled:cursor-not-allowed"
              onClick={() =>
                window.open('https://app.inngest.com/env/production/insights', '_blank')
              }
            >
              Use insights
            </button>
            <button
              className="border-muted text-btnPrimary bg-canvasBase focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:border-disabled disabled:bg-disabled disabled:text-btnPrimaryDisabled relative flex h-8 items-center justify-center justify-items-center whitespace-nowrap rounded-md border px-3 py-1.5 text-xs leading-[18px] disabled:cursor-not-allowed"
              onClick={() =>
                window.open('https://www.inngest.com/docs/platform/monitor/insights', '_blank')
              }
            >
              Read the docs
            </button>
          </div>
        </div>
      </div>
    )
  );
}
