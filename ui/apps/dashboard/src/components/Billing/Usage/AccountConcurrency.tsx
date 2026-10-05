import { Button } from '@inngest/components/Button';
import { Select } from '@inngest/components/Select/Select';
import Tabs from '@inngest/components/Tabs/Tabs';
import { useQuery } from 'urql';

import { GetBurstConcurrencyDocument } from '@/components/Billing/Usage/BurstConcurrency';
import { Route as BillingRoute } from '@/routes/_authed/billing/route';
import { pathCreator } from '@/utils/urls';

const ref = 'app-billing-usage-concurrency';
const periodOption = { id: '30d', name: '30 days' };

function formatBurstLimit(burstLimit: number, baseLimit?: number) {
  if (baseLimit && burstLimit % baseLimit === 0) {
    return `${burstLimit} (${baseLimit}×${burstLimit / baseLimit})`;
  }
  return `${burstLimit}`;
}

type Props = {
  baseLimit?: number;
};

export function AccountConcurrency({ baseLimit }: Props) {
  const { profile } = BillingRoute.useLoaderData();
  const [{ data, error }] = useQuery({ query: GetBurstConcurrencyDocument });
  const burstLimit = error ? undefined : data?.account.burstConcurrency?.limit;

  return (
    <div className="mt-10">
      <h3 className="text-basis mb-3 text-xl">Account concurrency</h3>
      <div className="mb-6 flex items-start justify-between gap-6">
        <p className="text-muted text-sm">
          The maximum number of steps your account can execute at the same time.
          Steps above this limit wait in the queue until capacity frees up.
        </p>
        <div className="flex shrink-0 gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
            label="Talk to sales"
            href={pathCreator.support({ ref })}
            target="_blank"
          />
          {!profile.isMarketplace && (
            <Button
              label="Increase concurrency"
              href={pathCreator.billing({ highlight: 'concurrency', ref })}
            />
          )}
        </div>
      </div>
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-center justify-between">
          <Tabs defaultValue="account" className="w-auto">
            <Tabs.List className="border-b-0">
              <Tabs.Tab value="account" title="Account concurrency" />
            </Tabs.List>
          </Tabs>
          <Select
            disabled
            onChange={() => {}}
            isLabelVisible
            label="Period"
            multiple={false}
            value={periodOption}
          >
            <Select.Button isLabelVisible size="small">
              <div className="text-sm font-medium">{periodOption.name}</div>
            </Select.Button>
          </Select>
        </div>
        <div className="mt-6 flex flex-wrap gap-10">
          {baseLimit !== undefined && (
            <div>
              <p className="text-basis text-sm">Concurrency limit</p>
              <p className="text-basis text-3xl">{baseLimit}</p>
            </div>
          )}
          {burstLimit !== undefined && (
            <div>
              <p className="text-basis text-sm">Burst limit</p>
              <p className="text-basis text-3xl">
                {formatBurstLimit(burstLimit, baseLimit)}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
