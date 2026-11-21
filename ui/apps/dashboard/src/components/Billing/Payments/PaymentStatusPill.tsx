import { Pill } from '@inngest/components/Pill';

type PaymentStatusPillProps = {
  // Stripe invoice status. Drafts are never listed.
  status: string;
};

export default function PaymentStatusPill({ status }: PaymentStatusPillProps) {
  let pill;
  switch (status) {
    case 'paid':
      pill = <Pill appearance="outlined">Paid</Pill>;
      break;
    case 'open':
      pill = (
        <Pill appearance="outlined" kind="warning">
          Awaiting payment
        </Pill>
      );
      break;
    case 'uncollectible':
      pill = (
        <Pill kind="error" appearance="outlined">
          Uncollectible
        </Pill>
      );
      break;
    case 'void':
      pill = <Pill appearance="outlined">Void</Pill>;
      break;
    default:
      pill = null;
  }

  return pill;
}
