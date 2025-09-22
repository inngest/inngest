'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiCloseLine } from '@remixicon/react';

const HIDE_FEATURE_LAUNCH_MON = 'inngest-feature-launch-mon-hide';

export default function FeaturelaunchMon() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Check if already dismissed
    if (window.localStorage.getItem(HIDE_FEATURE_LAUNCH_MON) === 'true') return;

    // Show the component (no date restrictions)
    setOpen(true);
  }, []);

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_FEATURE_LAUNCH_MON, 'true');
  };

  return (
    open && (
      <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[430px] rounded border">
        <div className="gap-x border-subtle flex flex-row items-center justify-between border-b p-3">
          <div className="text-sm font-medium leading-tight">
            Launch week day 1: Unbreakable APIs
          </div>
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
          Build unbreakable APIs using steps directly in your API endpoints, without the need for
          background workflows.
          <div className="flex gap-2 pt-2">
            <Button
              kind="primary"
              label="Learn more"
              href="https://www.inngest.com/blog/launch-week-day-1-unbreakable-apis?ref=launch-app-modal"
              target="_blank"
            />
            <Button
              label="Read the docs"
              appearance="outlined"
              href="https://www.inngest.com/docs/learn/rest-endpoints?ref=launch-app-modal"
              target="_blank"
            />
          </div>
        </div>
      </div>
    )
  );
}
