import { Link } from '@inngest/components/Link';
import { RiArrowRightLine, RiFlashlightFill } from '@remixicon/react';

import { ExecutionLimitPill } from '../ExecutionLimit/ExecutionLimitPill';
import { AlertPill } from '../Layout/AlertPill';
import { BURST_MULTIPLIER, minutesLeftLabel } from './concurrencyAlert';
import { useConcurrencyAlert } from './useConcurrencyAlert';

const linkClassName =
  'text-current text-xs font-black leading-4 hover:decoration-current';
const arrow = <RiArrowRightLine className="h-3.5 w-3.5" />;

export function ConcurrencyAlertPill() {
  const alert = useConcurrencyAlert();
  if (!alert) return <ExecutionLimitPill />;

  const linkProps = alert.linkProps('pill');

  if (alert.state === 'boosted') {
    return (
      <AlertPill
        kind="boost"
        icon={<RiFlashlightFill className="h-[18px] w-[18px]" />}
        action={
          alert.minutesRemaining != null && (
            <Link {...linkProps} className={linkClassName} iconAfter={arrow}>
              {minutesLeftLabel(alert.minutesRemaining)} left
            </Link>
          )
        }
      >
        Concurrency limit boosted by {BURST_MULTIPLIER}x.
      </AlertPill>
    );
  }

  return (
    <AlertPill
      kind="delayed"
      action={
        <Link
          {...linkProps}
          aria-label="Add concurrency"
          className={linkClassName}
          iconAfter={arrow}
        />
      }
    >
      Runs delayed because your concurrency limit was reached.
    </AlertPill>
  );
}
