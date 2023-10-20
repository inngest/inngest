'use client';

import { CheckIcon, ClockIcon, XMarkIcon } from '@heroicons/react/20/solid';
import * as Tooltip from '@radix-ui/react-tooltip';

type PaymentIconProps = {
  status: String;
};

export default function PaymentIcon({ status }: PaymentIconProps) {
  let icon;
  let label;
  switch (status) {
    case 'succeeded':
      icon = <CheckIcon className="mx-auto w-4 text-teal-500" />;
      label = 'Paid';
      break;
    case 'requires_payment_method':
      icon = <XMarkIcon className="mx-auto w-4 text-red-500" />;
      label = 'Failed';
      break;
    case 'canceled':
      icon = <XMarkIcon className="mx-auto w-4 text-slate-400" />;
      label = 'Canceled';
      break;
    case 'processing':
      icon = <ClockIcon className="mx-auto w-4 text-slate-500" />;
      label = 'Processing';
      break;
    default:
      icon = null;
      label = '';
  }
  if (icon) {
    return (
      <Tooltip.Provider>
        <Tooltip.Root delayDuration={0}>
          <Tooltip.Trigger asChild>{icon}</Tooltip.Trigger>
          <Tooltip.Content className="align-center rounded-md bg-slate-800 px-2 text-xs text-slate-300">
            {label}
            <Tooltip.Arrow className="fill-slate-800" />
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
    );
  }
  return icon;
}
