import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiArrowRightLine, RiErrorWarningFill } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { useExecutionLimit } from './useExecutionLimit';

export function ExecutionLimitPill() {
  const data = useExecutionLimit();
  if (!data?.isCapped) return null;

  return (
    <Pill
      action={
        <Link
          to={pathCreator.billing({
            tab: 'plans',
            ref: 'app-hobby-execution-limit-pill',
          })}
          className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense gap-0.5 text-xs leading-none"
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
      Hobby account execution limit reached. Upgrade your plan to continue using
      Inngest.
    </Pill>
  );
}
