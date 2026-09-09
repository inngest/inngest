import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiArrowRightLine, RiErrorWarningFill } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

export type ExecutionLimitPillProps = {
  className?: string;
  upgradeTo?: string;
};

export function ExecutionLimitPill({
  className,
  upgradeTo = pathCreator.billing({
    tab: 'plans',
    ref: 'app-hobby-execution-limit-pill',
  }),
}: ExecutionLimitPillProps) {
  return (
    <Pill
      action={
        <Link
          to={upgradeTo}
          className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense gap-0.5 text-xs leading-none"
          iconAfter={<RiArrowRightLine className="h-3.5 w-3.5" />}
        >
          Upgrade
        </Link>
      }
      appearance="outlined"
      className={className}
      icon={<RiErrorWarningFill className="h-3.5 w-3.5" />}
      iconSide="left"
      kind="error"
    >
      Hobby account execution limit reached. Upgrade your plan to continue using
      Inngest.
    </Pill>
  );
}
