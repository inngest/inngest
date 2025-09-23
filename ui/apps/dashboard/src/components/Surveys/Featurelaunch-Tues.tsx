'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { RiCloseLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

const HIDE_FEATURE_LAUNCH_TUES = 'inngest-feature-launch-tues-hide';

export default function FeaturelaunchTues({ envSlug }: { envSlug: string }) {
  const [open, setOpen] = useState(false);
  const router = useRouter();

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
          workflows run. No tool switching required.
          <div className="flex gap-2 pt-2">
            <Button
              label="Use insights"
              onClick={() => router.push(pathCreator.insights({ envSlug }))}
            />
            <Button
              label="Read the docs"
              appearance="outlined"
              href="https://www.inngest.com/docs/platform/monitor/insights?ref=launch-app-modal"
              target="_blank"
            />
          </div>
        </div>
      </div>
    )
  );
}
