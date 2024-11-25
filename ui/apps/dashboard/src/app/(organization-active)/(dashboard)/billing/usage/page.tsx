'use client';

import { useQuery } from 'urql';

import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { BillableStepUsage } from '../../settings/billing/BillableStepUsage/BillableStepUsage';

const GetBillingInfoDocument = graphql(`
  query GetBillingInfo {
    account {
      plan {
        id
        features
      }
    }
  }
`);

export default function Billing() {
  const [{ data, fetching }] = useQuery({
    query: GetBillingInfoDocument,
  });

  if (!data || fetching) {
    return (
      <div className="flex h-full min-h-[297px] w-full items-center justify-center overflow-hidden">
        <LoadingIcon />
      </div>
    );
  }

  let includedStepCountLimit: number | undefined;
  if (typeof data.account.plan?.features.actions === 'number') {
    includedStepCountLimit = data.account.plan.features.actions;
  }

  return <BillableStepUsage includedStepCountLimit={includedStepCountLimit} />;
}
