import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiArrowRightLine, RiErrorWarningFill } from '@remixicon/react';

import { AlertPill } from '../Layout/AlertPill';
import { upgradeLinkProps, useExecutionLimit } from './useExecutionLimit';

export function ExecutionLimitPill() {
  const data = useExecutionLimit();
  if (!data?.isCapped) return null;

  const linkProps = upgradeLinkProps(
    data.marketplaceBillingURL,
    'app-hobby-execution-limit-pill',
  );

  if (!data.enhanced) {
    return (
      <Pill
        action={
          <Link
            {...linkProps}
            className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense ml-4 gap-0.5 text-xs leading-none"
            iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
          >
            Upgrade
          </Link>
        }
        appearance="outlined"
        icon={<RiErrorWarningFill className="h-3.5 w-3.5" />}
        iconSide="left"
        kind="error"
      >
        Hobby account execution limit reached. Upgrade your plan to continue
        using Inngest.
      </Pill>
    );
  }

  return (
    <AlertPill
      action={
        <Link
          {...linkProps}
          className="text-alwaysWhite text-xs font-black leading-4 hover:decoration-current"
          iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
        >
          Upgrade
        </Link>
      }
    >
      {data.isVercel
        ? 'Hobby account executions limit reached. Change configurations on the Vercel Integrations Settings page to upgrade and resume using Inngest.'
        : 'Hobby account executions limit reached. Upgrade your plan to continue using Inngest.'}
    </AlertPill>
  );
}
