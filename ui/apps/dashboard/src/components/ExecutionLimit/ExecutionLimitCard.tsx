import { Link } from '@inngest/components/Link';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { RiArrowRightLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { SidebarAlertCard } from '../NavigationV2/SidebarAlertCard';

export type ExecutionLimitCardProps = {
  className?: string;
  executionLimit: number;
  upgradeTo?: string;
  usedExecutions: number;
};

const numberFormatter = new Intl.NumberFormat('en-US');

export function ExecutionLimitCard({
  className,
  executionLimit,
  upgradeTo = pathCreator.billing({
    tab: 'plans',
    ref: 'app-hobby-execution-limit-card',
  }),
  usedExecutions,
}: ExecutionLimitCardProps) {
  const formattedLimit = numberFormatter.format(executionLimit);
  const formattedUsage = numberFormatter.format(usedExecutions);

  return (
    <SidebarAlertCard
      className={className}
      kind="error"
      footer={
        <Link
          to={upgradeTo}
          className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense gap-0.5 text-xs leading-5"
          iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
        >
          Upgrade plan
        </Link>
      }
    >
      <h2 className="text-xs font-bold leading-4">New runs are paused</h2>
      <p className="mt-1 text-xs font-medium leading-4">
        You&apos;ve used {formattedUsage} executions in the last 30 days and
        reached the Hobby limit. New runs and scheduled functions won&apos;t
        start and aren&apos;t queued. Upgrade to Pro to resume now.
      </p>
      <div className="mt-2 pt-1">
        <ProgressBar
          kind="error"
          limit={executionLimit}
          size="small"
          value={usedExecutions}
        />
        <p className="mt-[3px] text-xs font-bold leading-4">
          {formattedUsage}/{formattedLimit} Executions
        </p>
      </div>
    </SidebarAlertCard>
  );
}
