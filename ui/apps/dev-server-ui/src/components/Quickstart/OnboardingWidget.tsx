import { Button } from '@inngest/components/Button';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { RiArrowRightLine, RiBookReadLine } from '@remixicon/react';
import { useQuickstart } from './QuickstartProvider';

export function OnboardingWidget({ collapsed }: { collapsed: boolean }) {
  const { completion, isDevServer, functions, ready } = useQuickstart();
  if (!ready || !isDevServer || completion) return null;
  if (collapsed)
    return (
      <MenuItem
        to="/?quickstart=true"
        collapsed
        text="Open quickstart"
        className="border-muted border"
        icon={
          <>
            <RiBookReadLine className="h-4 w-4" />
            <span className="sr-only">Open quickstart</span>
          </>
        }
      />
    );
  return (
    <div className="text-basis bg-canvasBase border-subtle mb-3 rounded border p-3 leading-tight">
      <img
        src="/images/onboarding-monitor.png"
        alt=""
        width={46}
        height={36}
        className="mb-3 h-9 w-auto"
        style={{ imageRendering: 'pixelated' }}
      />
      <p className="mb-1 font-medium">Inngest quickstart</p>
      <p className="text-subtle mb-4 text-sm">
        {functions.length
          ? 'Trigger your first successful function run.'
          : 'Connect an app and run your first function.'}
      </p>
      <Button
        to="/?quickstart=true"
        kind="primary"
        appearance="outlined"
        label="Get started"
        icon={<RiArrowRightLine />}
        iconSide="right"
      />
    </div>
  );
}
