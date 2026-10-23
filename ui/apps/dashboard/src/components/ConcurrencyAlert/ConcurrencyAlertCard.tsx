import { Link } from '@inngest/components/Link';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowRightLine,
  RiErrorWarningFill,
  RiRocket2Line,
} from '@remixicon/react';

import { ExecutionLimitCard } from '../ExecutionLimit/ExecutionLimitCard';
import { SidebarAlertCard } from '../NavigationV2/SidebarAlertCard';
import { BURST_MULTIPLIER, minutesLeftLabel } from './concurrencyAlert';
import {
  useConcurrencyAlert,
  type ConcurrencyAlert,
} from './useConcurrencyAlert';

const numberFormatter = new Intl.NumberFormat('en-US');

const arrow = <RiArrowRightLine className="h-3.5 w-3.5" />;

const boostedStyles = {
  card: 'bg-contrast border-contrast text-onContrast',
  divider: 'border-contrast',
  icon: 'text-secondary-moderate',
  link: 'text-secondary-subtle hover:decoration-secondary-subtle',
  track: 'bg-[rgb(var(--color-foreground-subtle))]',
  bar: 'bg-gradient-to-r from-secondary-moderate to-secondary-xSubtle',
};

const limitedStyles = {
  collapsed: 'border-accent-xSubtle bg-warning',
  icon: 'text-warning',
  link: 'text-warning decoration-warning hover:text-accent-2xIntense hover:decoration-accent-2xIntense',
};

export function ConcurrencyAlertCard({ collapsed }: { collapsed: boolean }) {
  const alert = useConcurrencyAlert();
  if (!alert) return <ExecutionLimitCard collapsed={collapsed} />;

  if (collapsed) {
    const boosted = alert.state === 'boosted';
    return (
      <MenuItem
        {...alert.linkProps('card-collapsed')}
        className={cn(
          'border',
          boosted ? boostedStyles.card : limitedStyles.collapsed,
        )}
        collapsed={collapsed}
        text={boosted ? 'View usage' : 'Add concurrency'}
        icon={
          boosted ? (
            <RiRocket2Line
              className={cn('h-[18px] w-[18px]', boostedStyles.icon)}
            />
          ) : (
            <RiErrorWarningFill
              className={cn('h-[18px] w-[18px]', limitedStyles.icon)}
            />
          )
        }
      />
    );
  }

  return alert.state === 'boosted' ? (
    <BoostedCard alert={alert} />
  ) : (
    <LimitedCard alert={alert} />
  );
}

function BoostedCard({ alert }: { alert: ConcurrencyAlert }) {
  return (
    <SidebarAlertCard
      className={cn('mb-5', boostedStyles.card)}
      footerClassName={boostedStyles.divider}
      footer={
        <Link
          {...alert.linkProps('card')}
          className={cn('gap-0.5 text-xs leading-5', boostedStyles.link)}
          iconAfter={arrow}
        >
          View usage
        </Link>
      }
    >
      <div className="flex items-center gap-1">
        <RiRocket2Line className={cn('h-4 w-4 shrink-0', boostedStyles.icon)} />
        <h2 className="text-xs font-bold leading-4">Concurrency boosted!</h2>
      </div>
      <p className="mt-1 text-xs font-medium leading-4">
        Your account concurrency has been boosted by {BURST_MULTIPLIER}x. Add
        concurrency to avoid delays after the boost ends.
      </p>
      <div className="mt-2 pt-1">
        <ProgressBar
          className={boostedStyles.track}
          indicatorClassName={boostedStyles.bar}
          limit={1}
          size="small"
          value={alert.usedRatio}
        />
        {alert.minutesRemaining != null && (
          <p className="mt-[3px] text-xs leading-4">
            {minutesLeftLabel(alert.minutesRemaining)} remaining this month
          </p>
        )}
      </div>
    </SidebarAlertCard>
  );
}

function LimitedCard({ alert }: { alert: ConcurrencyAlert }) {
  const formattedLimit =
    alert.limit != null ? numberFormatter.format(alert.limit) : null;

  return (
    <SidebarAlertCard
      className="border-accent-moderate mb-5"
      kind="warning"
      footer={
        <Link
          {...alert.linkProps('card')}
          className={cn('gap-0.5 text-xs leading-5', limitedStyles.link)}
          iconAfter={arrow}
        >
          Add concurrency
        </Link>
      }
    >
      <h2 className="text-xs font-bold leading-4">Concurrency limit reached</h2>
      <p className="mt-1 text-xs font-medium leading-4">
        Runs delayed due to your account hitting the concurrency limit. To speed
        up processing, increase your concurrency.
      </p>
      <div className="mt-2 pt-1">
        <ProgressBar
          indicatorClassName="bg-accent-xIntense"
          kind="warning"
          limit={1}
          size="small"
          value={1}
        />
        {formattedLimit != null && (
          <p className="mt-[3px] text-xs leading-4">
            {formattedLimit}/{formattedLimit} concurrent steps
          </p>
        )}
      </div>
    </SidebarAlertCard>
  );
}
