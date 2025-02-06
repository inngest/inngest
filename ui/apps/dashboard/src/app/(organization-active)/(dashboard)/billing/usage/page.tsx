'use client';

import { useState } from 'react';
import NextLink from 'next/link';
import { Select, type Option } from '@inngest/components/Select/Select';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';
import { useQuery } from 'urql';

import UsageMetadata from '@/components/Billing/Usage/Metadata';
import UsageChart from '@/components/Billing/Usage/UsageChart';
import useGetUsageChartData from '@/components/Billing/Usage/useGetUsageChartData';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

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

export default function Billing({
  searchParams: { previous },
}: {
  searchParams: { previous: boolean };
}) {
  const [{ data, fetching }] = useQuery({
    query: GetBillingInfoDocument,
  });

  const [currentPage, setCurrentPage] = useState<'run' | 'step'>('step');
  const [selectedPeriod, setSelectedPeriod] = useState<Period>(previous ? options[1] : options[0]);

  // Get timeseries to temporarily grab the total usage for previous month, since we don't have history usage on entitlements
  const { data: billableData, fetching: fetchingBillableData } = useGetUsageChartData({
    selectedPeriod: selectedPeriod.id,
    type: currentPage,
  });

  const stepCount = data?.account.entitlements.stepCount || { usage: 0, limit: 0 };
  const runCount = data?.account.entitlements.runCount || { usage: 0, limit: 0 };
  const isStepPage = currentPage === 'step';
  const currentUsage = (() => {
    if (selectedPeriod.id === 'previous' && billableData.length) {
      return billableData.reduce((sum, point) => sum + (point.value || 0), 0);
    }
    return isStepPage ? stepCount.usage : runCount.usage;
  })();
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
          onValueChange={(value) => {
            if (value === 'run' || value === 'step') {
              setCurrentPage(value);
            }
          }}
        >
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
            <div className="text-basis text-sm font-medium">{selectedPeriod.name}</div>
          </Select.Button>
          <Select.Options>
            <NextLink href={pathCreator.billing({ tab: 'usage' })}>
              <Select.Option option={options[0]}>{options[0].name}</Select.Option>
            </NextLink>
            <NextLink href={pathCreator.billing({ tab: 'usage' }) + '?previous=true'}>
              <Select.Option option={options[1]}>{options[1].name}</Select.Option>
            </NextLink>
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
}
