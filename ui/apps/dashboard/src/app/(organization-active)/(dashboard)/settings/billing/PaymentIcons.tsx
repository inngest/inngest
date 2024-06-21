'use client';

import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiCheckLine, RiCloseLine, RiErrorWarningLine, RiTimeLine } from '@remixicon/react';

type PaymentIconProps = {
  status: String;
};

export default function PaymentIcon({ status }: PaymentIconProps) {
  let icon;
  let label;
  switch (status) {
    case 'succeeded':
      icon = <RiCheckLine className="mx-auto w-4 text-teal-500" />;
      label = 'Paid';
      break;
    case 'requires_payment_method':
      icon = <RiCloseLine className="mx-auto w-4 text-red-500" />;
      label = 'Failed';
      break;
    case 'canceled':
      icon = <RiCloseLine className="mx-auto w-4 text-slate-400" />;
      label = 'Canceled';
      break;
    case 'processing':
      icon = <RiTimeLine className="mx-auto w-4 text-slate-500" />;
      label = 'Processing';
      break;
    case 'requires_confirmation':
      icon = <RiErrorWarningLine className="mx-auto w-4 text-amber-500" />;
      label = 'Awaiting payment';
      break;
    default:
      icon = null;
      label = '';
  }
  if (icon) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{icon}</TooltipTrigger>
        <TooltipContent className="align-center rounded-md px-2 text-xs">{label}</TooltipContent>
      </Tooltip>
    );
  }
  return icon;
}
