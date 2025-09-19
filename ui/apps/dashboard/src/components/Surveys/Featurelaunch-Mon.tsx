'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link';
import { RiCloseLine } from '@remixicon/react';

const HIDE_FEATURE_LAUNCH = 'inngest-feature-launch-hide';

export default function FeaturelaunchMon() {
  const [mounted, setMounted] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    setMounted(true);

    if (typeof window === 'undefined') return;

    // Check if already dismissed
    if (window.localStorage.getItem(HIDE_FEATURE_LAUNCH) === 'true') return;

    // Check if today is Monday, September 22nd
    const today = new Date();
    const targetDate = new Date(2025, 8, 22); // Month is 0-indexed, so 8 = September

    // Check if it's the same date (year, month, day)
    const isTargetDate =
      today.getFullYear() === targetDate.getFullYear() &&
      today.getMonth() === targetDate.getMonth() &&
      today.getDate() === targetDate.getDate();

    if (isTargetDate) {
      setOpen(true);
    }
  }, []);

  const dismiss = () => {
    setOpen(false);
    window.localStorage.setItem(HIDE_FEATURE_LAUNCH, 'true');
  };

  if (!mounted) return null;

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
          background workflows
          <div className="pt-2">
            <button
              className="border-muted text-btnPrimary bg-canvasBase focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted disabled:border-disabled disabled:bg-disabled disabled:text-btnPrimaryDisabled relative flex h-8 items-center justify-center justify-items-center whitespace-nowrap rounded-md border px-3 py-1.5 text-xs leading-[18px] disabled:cursor-not-allowed"
              onClick={() => window.open('https://www.inngest.com/blog', '_blank')}
            >
              Read the blog post
            </button>
          </div>
        </div>
      </div>
    )
  );
}
