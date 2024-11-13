import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import { LimitBar } from '@/components/Billing/LimitBars/LimitBar';
import { pathCreator } from '@/utils/urls';

const plan = 'Free';
const runs = {
  title: 'Runs',
  description: 'A single durable function execution. Additional runs are $5 per 200k.',
  current: 8000000,
  limit: 5000000,
};

const steps = {
  title: 'Steps',
  description: 'An individual step in durable functions. Additional steps are $5 per 200k.',
  current: 15000000,
  limit: 25000000,
};
export default function Page() {
  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        <Card.Content>
          <p className="text-muted mb-1">Your plan</p>
          <p className="text-basis text-xl">{plan}</p>
          <LimitBar data={runs} className="my-4" />
          <LimitBar data={steps} className="mb-6" />
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
