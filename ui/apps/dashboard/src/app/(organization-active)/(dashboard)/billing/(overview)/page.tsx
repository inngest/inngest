import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import BillingInformation from '@/components/Billing/BillingDetails/BillingInformation';
import PaymentMethod from '@/components/Billing/BillingDetails/PaymentMethod';
import { LimitBar } from '@/components/Billing/LimitBar';
import {
  getBillingDetails,
  getCurrentPlan,
  getEntitlementUsage,
} from '@/components/Billing/actions';
import { day } from '@/utils/date';
import { pathCreator } from '@/utils/urls';

export default async function Page() {
  const entitlementUsage = await getEntitlementUsage();
  const plan = await getCurrentPlan();
  const billing = await getBillingDetails();

  const runs = {
    title: 'Runs',
    description: `A single durable function execution. ${
      entitlementUsage?.runCount.overageAllowed ? 'Additional runs are available for purchase.' : ''
    }`,
    current: entitlementUsage?.runCount.current || 0,
    limit: entitlementUsage?.runCount.limit || null,
  };

  const steps = {
    title: 'Steps',
    description: `An individual step in durable functions. ${
      entitlementUsage?.runCount.overageAllowed
        ? 'Additional steps are available for purchase.'
        : ''
    }`,
    current: entitlementUsage?.stepCount.current || 0,
    limit: entitlementUsage?.stepCount.limit || null,
  };

  const nextInvoiceDate = plan?.subscription?.nextInvoiceDate
    ? day(plan.subscription.nextInvoiceDate)
    : undefined;

  const nextInvoiceAmount = plan?.plan?.amount ? `$${plan.plan.amount / 100}` : 'Free';

  const paymentMethod = billing?.paymentMethods?.[0] || null;
  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        <Card.Content>
          <p className="text-muted mb-1">Your plan</p>
          <div className="flex items-center justify-between">
            <p className="text-basis text-xl">{plan?.plan?.name}</p>
            {/* Temporarily send to usage, while there is no plans page */}
            <NewButton
              appearance="ghost"
              label="Change plan"
              href="/billing/usage?ref=app-billing-overview"
            />
          </div>
          {entitlementUsage?.runCount && <LimitBar data={runs} className="my-4" />}
          {entitlementUsage?.stepCount && <LimitBar data={steps} className="mb-6" />}
          <div className="flex flex-col items-center gap-2">
            <p>Custom needs?</p>
            <NewButton
              appearance="outlined"
              label="Chat with a product expert"
              href={pathCreator.support({ ref: 'app-billing-overview' })}
            />
          </div>
        </Card.Content>
      </Card>
      <div className="col-span-1">
        <Card className="mb-4">
          <Card.Content>
            <p className="text-muted mb-1">Next payment</p>
            <p className="text-basis text-lg">{nextInvoiceAmount}</p>
            {nextInvoiceDate && (
              <>
                <p className="text-subtle mb-1 mt-4 text-xs font-medium">Payment due date</p>
                <p className="text-basis text-sm">{nextInvoiceDate}</p>
              </>
            )}
          </Card.Content>
        </Card>
        {billing && (
          <BillingInformation billingEmail={billing.billingEmail} accountName={billing.name} />
        )}
        <PaymentMethod paymentMethod={paymentMethod} />
      </div>
    </div>
  );
}
