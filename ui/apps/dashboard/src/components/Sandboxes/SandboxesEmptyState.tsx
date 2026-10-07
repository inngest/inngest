import { useEffect, useRef } from 'react';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { RiArrowRightLine, RiLockLine } from '@remixicon/react';

import {
  trackEmptyStateUpgradeClicked,
  trackEmptyStateViewed,
} from '@/utils/analyticsEvents';
import { pathCreator } from '@/utils/urls';

import {
  FEATURES,
  INTRO_LEAD,
  INTRO_REST,
  USE_CASES,
} from './sandboxesContent';

const UPGRADE_HREF = pathCreator.billing({
  tab: 'plans',
  ref: 'app-sandboxes-empty-state',
});

function UpgradeButton() {
  return (
    <Button
      kind="primary"
      label="Upgrade plan"
      icon={<RiArrowRightLine />}
      iconSide="right"
      href={UPGRADE_HREF}
      onClick={() => trackEmptyStateUpgradeClicked({ feature: 'sandboxes' })}
    />
  );
}

// Shown to accounts whose plan does not grant sandbox access.
export default function SandboxesEmptyState() {
  const hasTrackedViewed = useRef(false);

  // Fire the feature-tagged page-view event once per mount.
  useEffect(() => {
    if (hasTrackedViewed.current) return;
    hasTrackedViewed.current = true;
    trackEmptyStateViewed({ feature: 'sandboxes' });
  }, []);

  return (
    <div className="mx-auto w-full max-w-[816px] px-6 py-16">
      <div className="flex items-center gap-2">
        <h1 className="text-basis text-2xl">Sandboxes</h1>
        <Pill kind="primary" appearance="outlined">
          Paid plans
        </Pill>
      </div>
      <p className="text-subtle mt-2 text-sm leading-relaxed">
        <span className="text-basis font-medium">{INTRO_LEAD}</span>
        {INTRO_REST}
      </p>

      <div className="border-subtle bg-canvasSubtle mt-6 flex flex-col gap-4 rounded-md border p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="border-subtle bg-canvasBase text-muted flex h-9 w-9 shrink-0 items-center justify-center rounded-md border">
            <RiLockLine className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-basis text-base">
              Sandboxes are available on paid plans
            </h2>
            <p className="text-muted mt-0.5 text-sm">
              Upgrade to run sandboxes from your workflows, including from the
              local dev server.
            </p>
          </div>
        </div>
        <div className="shrink-0">
          <UpgradeButton />
        </div>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        {FEATURES.map(({ Icon, title, description }) => (
          <div
            key={title}
            className="border-subtle bg-canvasBase rounded-md border p-3"
          >
            <div className="text-muted flex h-9 w-9 items-center justify-center">
              <Icon className="h-6 w-6" />
            </div>
            <h3 className="text-basis mt-3 text-base">{title}</h3>
            <p className="text-muted mt-1 text-xs leading-relaxed">
              {description}
            </p>
          </div>
        ))}
      </div>

      <h2 className="text-basis mt-10 text-base">What can you build?</h2>
      <div className="mt-4 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
        {USE_CASES.map(({ Icon, title, description }) => (
          <div key={title} className="flex items-start gap-3">
            <div className="border-subtle text-muted flex h-[46px] w-[46px] shrink-0 items-center justify-center rounded-md border">
              <Icon className="h-6 w-6" />
            </div>
            <div>
              <h3 className="text-basis text-base">{title}</h3>
              <p className="text-muted mt-0.5 text-xs">{description}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-10">
        <UpgradeButton />
      </div>
    </div>
  );
}
