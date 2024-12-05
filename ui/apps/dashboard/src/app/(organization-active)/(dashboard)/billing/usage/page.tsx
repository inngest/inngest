'use client';

import { useState } from 'react';
import { Select, type Option } from '@inngest/components/Select/Select';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';
import { useQuery } from 'urql';

import { BillableUsageChart } from '@/components/Billing/Usage/BillableUsageChart';
import UsageMetadata from '@/components/Billing/Usage/Metadata';
import { graphql } from '@/gql';

const GetBillingInfoDocument = graphql(`
  query GetBillingInfo {
    account {
      entitlements {
        stepCount {
          usage
          limit
        }
        runCount {
          usage
          limit
        }
      }
    }
  }
`);

type Period = Option & { id: 'current' | 'previous' };

const options: Period[] = [
  {
    id: 'current',
    name: 'This month',
  },
  {
    id: 'previous',
    name: 'Last month',
  },
];

export default function Billing() {
  const [{ data, fetching }] = useQuery({
    query: GetBillingInfoDocument,
  });

  const [currentPage, setCurrentPage] = useState('step');
  const [selectedPeriod, setSelectedPeriod] = useState<Period>(options[0]!);

  const stepCount = data?.account.entitlements.stepCount || { usage: 0, limit: 0 };
  const runCount = data?.account.entitlements.runCount || { usage: 0, limit: 0 };
  const isStepPage = currentPage === 'step';
  const currentUsage = isStepPage ? stepCount.usage : runCount.usage;
  const currentLimit = isStepPage ? stepCount.limit ?? Infinity : runCount.limit ?? Infinity;
  const additionalUsage = Math.max(0, currentUsage - currentLimit);

  function isPeriod(option: Option): option is Period {
    return ['current', 'previous'].includes(option.id);
  }

  return (
    <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
      <div className="flex items-center justify-between">
        <ToggleGroup
          type="single"
          defaultValue={currentPage}
          size="small"
          onValueChange={setCurrentPage}
        >
          <ToggleGroup.Item value="run">Run</ToggleGroup.Item>
          <ToggleGroup.Item value="step">Step</ToggleGroup.Item>
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
            <div className="text-basis text-sm font-medium">{selectedPeriod.name}</div>
          </Select.Button>
          <Select.Options>
            <Select.Option option={options[0]!}>{options[0]!.name}</Select.Option>
            <Select.Option option={options[1]!}>{options[1]!.name}</Select.Option>
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
          fetching={fetching}
          title={`Total ${currentPage}s`}
          value={new Intl.NumberFormat().format(currentUsage)}
        />
      </dl>
      {isStepPage && (
        <BillableUsageChart
          selectedPeriod={selectedPeriod.id}
          includedStepCountLimit={currentLimit}
          type={currentPage}
        />
      )}
    </div>
  );
}
