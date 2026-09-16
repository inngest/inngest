import { Link } from '@inngest/components/Link';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { RiArrowRightLine, RiErrorWarningFill } from '@remixicon/react';

import { SidebarAlertCard } from '../NavigationV2/SidebarAlertCard';
import { upgradeLinkProps, useExecutionLimit } from './useExecutionLimit';

const numberFormatter = new Intl.NumberFormat('en-US');

export function ExecutionLimitCard({ collapsed }: { collapsed: boolean }) {
  const data = useExecutionLimit();
  if (!data?.isCapped) return null;

  if (collapsed) {
    return (
      <MenuItem
        {...upgradeLinkProps(
          data.marketplaceBillingURL,
          'app-hobby-execution-limit-card-collapsed',
        )}
        className="border-tertiary-xSubtle bg-error border"
        collapsed={collapsed}
        text="Upgrade plan"
        icon={<RiErrorWarningFill className="text-error h-[18px] w-[18px]" />}
      />
    );
  }

  const formattedLimit = numberFormatter.format(data.executionLimit);
  const formattedUsage = numberFormatter.format(data.usedExecutions);

  return (
    <SidebarAlertCard
      className="mb-5"
      kind="error"
      footer={
        <Link
          {...upgradeLinkProps(
            data.marketplaceBillingURL,
            'app-hobby-execution-limit-card',
          )}
          className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense gap-0.5 text-xs leading-5"
          iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
        >
          Upgrade plan
        </Link>
      }
    >
      <h2 className="text-xs font-bold leading-4">Execution Limit Reached</h2>
      <p className="mt-1 text-xs font-medium leading-4">
        You&apos;ve used {formattedUsage} executions this month and reached the
        Hobby limit. Performance will be impacted. Upgrade to Pro to avoid
        disruption.
      </p>
      <div className="mt-2 pt-1">
        <ProgressBar
          kind="error"
          limit={data.executionLimit}
          size="small"
          value={data.usedExecutions}
        />
        <p className="mt-[3px] text-xs font-bold leading-4">
          {formattedUsage}/{formattedLimit} Executions
        </p>
      </div>
    </SidebarAlertCard>
  );
}
