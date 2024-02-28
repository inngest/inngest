'use client';

import { CheckIcon, ClockIcon, ExclamationCircleIcon, XMarkIcon } from '@heroicons/react/20/solid';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

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
    case 'requires_confirmation':
      icon = <ExclamationCircleIcon className="mx-auto w-4 text-amber-500" />;
      label = 'Awaiting payment';
      break;
    default:
      icon = null;
      label = '';
  }
  if (icon) {
    return (
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>{icon}</TooltipTrigger>
        <TooltipContent className="align-center rounded-md bg-slate-800 px-2 text-xs text-slate-300">
          {label}
        </TooltipContent>
      </Tooltip>
    );
  }
  return icon;
}
