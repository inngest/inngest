import { useState } from 'react';
import { Select, type Option } from '@inngest/components/Select/Select';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';
import { useQuery } from 'urql';

import UsageChart from '@/components/Billing/Usage/UsageChart';
import {
  type UsageDimension,
  isUsageDimension,
} from '@/components/Billing/Usage/types';
import { graphql } from '@/gql';
import { Route as BillingUsageRoute } from '@/routes/_authed/billing/usage/index';

const GetBillingInfoDocument = graphql(`
  query GetBillingInfo {
    account {
      entitlements {
        executions {
          limit
        }
        stepCount {
          limit
        }
        runCount {
          limit
        }
      }
      plan {
        slug
      }
    }
  }
`);

type Period = Option & { id: 'current' | 'previous' };

const options = [
  {
    id: 'current',
    name: 'This month',
  },
  {
    id: 'previous',
    name: 'Last month',
  },
] as const satisfies Readonly<Period[]>;

type UsagePageProps = {
  previous?: boolean;
  dimension?: UsageDimension;
};

export const UsagePage = ({
  previous,
  dimension = 'execution',
}: UsagePageProps) => {
  const navigate = BillingUsageRoute.useNavigate();
  const [{ data }] = useQuery({
    query: GetBillingInfoDocument,
  });

  // const [currentPage, setCurrentPage] = useState<UsageDimension>(dimension);
  const [selectedPeriod, setSelectedPeriod] = useState<Period>(
    previous ? options[1] : options[0],
  );

  let currentLimit = Infinity;
  if (data) {
    if (dimension === 'execution') {
      currentLimit = data.account.entitlements.executions.limit ?? Infinity;
    } else if (dimension === 'run') {
      currentLimit = data.account.entitlements.runCount.limit ?? Infinity;
    } else {
      currentLimit = data.account.entitlements.stepCount.limit ?? Infinity;
    }
  }

  const isPeriod = (option: Option): option is Period => {
    return ['current', 'previous'].includes(option.id);
  };

  const dimensionTitle = dimension.charAt(0).toUpperCase() + dimension.slice(1);

  return (
    <>
      <h3 className="text-basis mb-4 text-xl">{dimensionTitle}</h3>
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-center justify-between">
          <ToggleGroup
            type="single"
            defaultValue={dimension}
            size="small"
            onValueChange={(value) => {
              if (!isUsageDimension(value)) {
                console.error('invalid usage dimension', value);
                return;
              }
              navigate({
                search: {
                  dimension: value,
                  ...(previous && { previous }),
                },
              });
            }}
          >
            <ToggleGroup.Item value="execution">Executions</ToggleGroup.Item>
            <ToggleGroup.Item value="run">Runs</ToggleGroup.Item>
            <ToggleGroup.Item value="step">Steps</ToggleGroup.Item>
          </ToggleGroup>
          <Select
            onChange={(value: Option) => {
              if (isPeriod(value)) {
                setSelectedPeriod(value);
              }
            }}
            isLabelVisible
            label="Billing period"
            multiple={false}
            value={selectedPeriod}
          >
            <Select.Button isLabelVisible size="small">
              <div className="text-basis text-sm font-medium">
                {selectedPeriod.name}
              </div>
            </Select.Button>
            <Select.Options>
              <BillingUsageRoute.Link search={{ dimension }}>
                <Select.Option option={options[0]}>
                  {options[0].name}
                </Select.Option>
              </BillingUsageRoute.Link>
              <BillingUsageRoute.Link search={{ dimension, previous: true }}>
                <Select.Option option={options[1]}>
                  {options[1].name}
                </Select.Option>
              </BillingUsageRoute.Link>
            </Select.Options>
          </Select>
        </div>
        <div className="mt-6">
          <UsageChart
            selectedPeriod={selectedPeriod.id}
            includedCountLimit={currentLimit}
            type={dimension}
          />
        </div>
      </div>
    </>
  );
};
