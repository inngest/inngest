'use client';

import { Pill } from '@inngest/components/Pill/Pill';

type PaymentStatusPillProps = {
  status: String;
};

export default function PaymentStatusPill({ status }: PaymentStatusPillProps) {
  let pill;
  switch (status) {
    case 'succeeded':
      pill = <Pill appearance="outlined">Paid</Pill>;
      break;
    case 'requires_payment_method':
      pill = (
        <Pill kind="error" appearance="outlined">
          Failed
        </Pill>
      );
      break;
    case 'canceled':
      pill = <Pill appearance="outlined">Canceled</Pill>;
      break;
    case 'processing':
      pill = <Pill appearance="outlined">Processing</Pill>;
      break;
    case 'requires_confirmation':
      pill = (
        <Pill appearance="outlined" kind="warning">
          Awaiting payment
        </Pill>
      );
      break;
    default:
      pill = null;
  }

  return pill;
}
