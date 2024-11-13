import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import { LimitBar } from '@/components/Billing/LimitBar';
import { getEntitlementUsage } from '@/components/Billing/actions';
import { pathCreator } from '@/utils/urls';

const plan = 'Free';

export default async function Page() {
  const entitlementUsage = await getEntitlementUsage();

  const runs = {
    title: 'Runs',
    description: 'A single durable function execution. Additional runs are available for purchase.',
    current: entitlementUsage?.runCount?.current || 0,
    limit: entitlementUsage?.runCount?.limit || 0,
  };

  // const steps = {
  //   title: 'Steps',
  //   description: 'An individual step in durable functions. Additional steps are available for purchase.',
  //   current: entitlementUsage?.stepCount?.current || 0,
  //   limit: entitlementUsage?.stepCount?.limit || 0,
  // };

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        <Card.Content>
          <p className="text-muted mb-1">Your plan</p>
          <p className="text-basis text-xl">{plan}</p>
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
          <p className="text-basis text-lg">{plan}</p>
        </Card.Content>
      </Card>
    </div>
  );
}
