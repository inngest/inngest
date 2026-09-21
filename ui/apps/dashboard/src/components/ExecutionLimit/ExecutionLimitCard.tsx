import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowRightLine,
  RiCloseLine,
  RiErrorWarningFill,
} from '@remixicon/react';

import { SidebarAlertCard } from '../NavigationV2/SidebarAlertCard';
import type { UsageBand } from './executionLimit';
import { useDismissal } from './useDismissal';
import { upgradeLinkProps, useExecutionLimit } from './useExecutionLimit';

const numberFormatter = new Intl.NumberFormat('en-US');

const kindStyles = {
  error: {
    collapsed: 'border-tertiary-xSubtle bg-error',
    icon: 'text-error',
    link: 'text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense',
  },
  warning: {
    collapsed: 'border-accent-xSubtle bg-warning',
    icon: 'text-warning',
    link: 'text-warning decoration-warning hover:text-accent-2xIntense hover:decoration-accent-2xIntense',
  },
  caution: {
    collapsed: 'border-accent-2xSubtle bg-canvasSubtle',
    icon: 'text-accent-moderate',
    link: 'text-accent-intense decoration-accent-intense hover:text-accent-2xIntense hover:decoration-accent-2xIntense',
  },
} as const;

type CardContent = {
  kind: keyof typeof kindStyles;
  title: string;
  body: (usage: string) => string;
  dismissable: boolean;
};

const legacyContent: CardContent = {
  kind: 'error',
  title: 'Execution Limit Reached',
  body: (usage) =>
    `You've used ${usage} executions this month and reached the Hobby limit. Performance will be impacted. Upgrade to Pro to avoid disruption.`,
  dismissable: false,
};

const pausedContent = {
  title: 'New runs will be paused',
  body: (usage: string) =>
    `You've used ${usage} executions this month and are close to the Hobby limit. New runs and scheduled functions will stop and won't be queued. Upgrade to Pro to resume now.`,
  dismissable: true,
};

const enhancedContent: Partial<Record<UsageBand, CardContent>> = {
  '50': { kind: 'caution', ...pausedContent },
  '75': { kind: 'warning', ...pausedContent },
  '90': { kind: 'error', ...pausedContent },
  capped: {
    kind: 'error',
    title: 'New runs are paused',
    body: (usage) =>
      `You've used ${usage} executions this month and reached the Hobby limit. New runs and scheduled functions won't start and aren't queued. Upgrade to Pro to resume now.`,
    dismissable: false,
  },
};

export function ExecutionLimitCard({ collapsed }: { collapsed: boolean }) {
  const data = useExecutionLimit();
  const { isReady, isDismissed, dismiss } = useDismissal(
    'card',
    data?.accountID,
  );
  if (!data) return null;

  let content: CardContent | undefined;
  if (data.enhanced) {
    content = enhancedContent[data.band];
  } else if (data.isCapped) {
    content = legacyContent;
  }
  if (!content) return null;
  if (content.dismissable && (!isReady || isDismissed)) return null;

  const styles = kindStyles[content.kind];

  if (collapsed) {
    return (
      <MenuItem
        {...upgradeLinkProps(
          data.marketplaceBillingURL,
          'app-hobby-execution-limit-card-collapsed',
        )}
        className={cn('border', styles.collapsed)}
        collapsed={collapsed}
        text="Upgrade plan"
        icon={
          <RiErrorWarningFill
            className={cn('h-[18px] w-[18px]', styles.icon)}
          />
        }
      />
    );
  }

  const formattedLimit = numberFormatter.format(data.executionLimit);
  const formattedUsage = numberFormatter.format(data.usedExecutions);

  return (
    <SidebarAlertCard
      className="mb-5"
      kind={content.kind}
      footer={
        <Link
          {...upgradeLinkProps(
            data.marketplaceBillingURL,
            'app-hobby-execution-limit-card',
          )}
          className={cn('gap-0.5 text-xs leading-5', styles.link)}
          iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
        >
          Upgrade plan
        </Link>
      }
    >
      {content.dismissable ? (
        <div className="flex items-start justify-between gap-1">
          <h2 className="text-xs font-bold leading-4">{content.title}</h2>
          <Button
            icon={<RiCloseLine className={styles.icon} />}
            kind="secondary"
            appearance="ghost"
            size="small"
            className="-mr-1 -mt-1 shrink-0"
            tooltip="Dismiss for 24 hours"
            aria-label="Dismiss for 24 hours"
            onClick={dismiss}
          />
        </div>
      ) : (
        <h2 className="text-xs font-bold leading-4">{content.title}</h2>
      )}
      <p className="mt-1 text-xs font-medium leading-4">
        {content.body(formattedUsage)}
      </p>
      <div className="mt-2 pt-1">
        <ProgressBar
          kind={content.kind}
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
