import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import { LimitBar } from '@/components/Billing/LimitBar';
import { getCurrentPlan, getEntitlementUsage } from '@/components/Billing/actions';
import { pathCreator } from '@/utils/urls';

export default async function Page() {
  const entitlementUsage = await getEntitlementUsage();
  const account = await getCurrentPlan();

  const runs = {
    title: 'Runs',
    description: 'A single durable function execution. Additional runs are available for purchase.',
    current: entitlementUsage?.runCount.current || 0,
    limit: entitlementUsage?.runCount.limit || null,
  };

  // const steps = {
  //   title: 'Steps',
  //   description: 'An individual step in durable functions. Additional steps are available for purchase.',
  //   current: entitlementUsage?.stepCount?.current || 0,
  //   limit: entitlementUsage?.stepCount?.limit || null,
  // };

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        <Card.Content>
          <p className="text-muted mb-1">Your plan</p>
          <p className="text-basis text-xl">{account?.plan?.name}</p>
          {entitlementUsage?.runCount && <LimitBar data={runs} className="my-4" />}
          {/* {entitlementUsage?.stepCount && <LimitBar data={steps} className="mb-6" />} */}
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
      <Card className="col-span-1">
        <Card.Content>
          <p className="text-muted mb-1">Next payment</p>
          <p className="text-basis text-lg">{account?.subscription?.nextInvoiceDate}</p>
        </Card.Content>
      </Card>
    </div>
  );
}