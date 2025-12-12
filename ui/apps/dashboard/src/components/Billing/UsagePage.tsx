import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Select, type Option } from '@inngest/components/Select/NewSelect';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';
import { useQuery } from 'urql';

import UsageMetadata from '@/components/Billing/Usage/Metadata';
import UsageChart from '@/components/Billing/Usage/UsageChart';
import {
  isUsageDimension,
  type UsageDimension,
} from '@/components/Billing/Usage/types';
import useGetUsageChartData from '@/components/Billing/Usage/useGetUsageChartData';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

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
};

export const UsagePage = ({ previous }: UsagePageProps) => {
  const [{ data, fetching }] = useQuery({
    query: GetBillingInfoDocument,
  });

  const [currentPage, setCurrentPage] = useState<UsageDimension>('execution');
  const [selectedPeriod, setSelectedPeriod] = useState<Period>(
    previous ? options[1] : options[0],
  );

  //
  // Get timeseries to temporarily grab the total usage for previous month, since we don't have history usage on entitlements
  const { data: billableData, fetching: fetchingBillableData } =
    useGetUsageChartData({
      selectedPeriod: selectedPeriod.id,
      type: currentPage,
    });

  const currentUsage = billableData.reduce((sum, point) => {
    return sum + (point.value || 0);
  }, 0);

  let currentLimit = Infinity;
  if (data) {
    if (currentPage === 'execution') {
      currentLimit = data.account.entitlements.executions.limit ?? Infinity;
    } else if (currentPage === 'run') {
      currentLimit = data.account.entitlements.runCount.limit ?? Infinity;
    } else {
      currentLimit = data.account.entitlements.stepCount.limit ?? Infinity;
    }
  }

  const additionalUsage = Math.max(0, currentUsage - currentLimit);

  const isPeriod = (option: Option): option is Period => {
    return ['current', 'previous'].includes(option.id);
  };

  return (
    <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
      <div className="flex items-center justify-between">
        <ToggleGroup
          type="single"
          defaultValue={currentPage}
          size="small"
          onValueChange={(value) => {
            if (!isUsageDimension(value)) {
              console.error('invalid usage dimension', value);
              return;
            }
            setCurrentPage(value);
          }}
        >
          <ToggleGroup.Item value="execution">Execution</ToggleGroup.Item>
          <ToggleGroup.Item value="step">Step</ToggleGroup.Item>
          <ToggleGroup.Item value="run">Run</ToggleGroup.Item>
        </ToggleGroup>
        <Select
          onChange={(value: Option) => {
            if (isPeriod(value)) {
              setSelectedPeriod(value);
            }
          }}
          isLabelVisible
          label="Period"
          multiple={false}
          value={selectedPeriod}
        >
          <Select.Button isLabelVisible size="small">
            <div className="text-basis text-sm font-medium">
              {selectedPeriod.name}
            </div>
          </Select.Button>
          <Select.Options>
            <Link to={pathCreator.billing({ tab: 'usage' })}>
              <Select.Option option={options[0]}>
                {options[0].name}
              </Select.Option>
            </Link>
            <Link to={pathCreator.billing({ tab: 'usage' }) + '?previous=true'}>
              <Select.Option option={options[1]}>
                {options[1].name}
              </Select.Option>
            </Link>
          </Select.Options>
        </Select>
      </div>
      <dl className="my-6 grid grid-cols-3">
        <UsageMetadata
          className="justify-self-start"
          fetching={fetching}
          title={`Plan-included ${currentPage}s`}
          value={new Intl.NumberFormat().format(currentLimit)}
        />
        <UsageMetadata
          className="justify-self-center"
          fetching={fetching}
          title={`Additional ${currentPage}s`}
          value={new Intl.NumberFormat().format(additionalUsage)}
        />
        <UsageMetadata
          className="justify-self-end"
          fetching={fetching || fetchingBillableData}
          title={`Total ${currentPage}s`}
          value={new Intl.NumberFormat().format(currentUsage)}
        />
      </dl>
      <UsageChart
        selectedPeriod={selectedPeriod.id}
        includedCountLimit={currentLimit}
        type={currentPage}
      />
    </div>
  );
};
