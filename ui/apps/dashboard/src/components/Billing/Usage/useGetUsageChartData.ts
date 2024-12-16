import { useQuery } from 'urql';

import { graphql } from '@/gql';
import {
  type GetBillableRunsQuery,
  type GetBillableStepsQuery,
  type TimeSeriesPoint,
} from '@/gql/graphql';

const GetBillableSteps = graphql(`
  query GetBillableSteps($month: Int!, $year: Int!) {
    billableStepTimeSeries(timeOptions: { month: $month, year: $year }) {
      data {
        time
        value
      }
    }
  }
`);

const GetBillableRuns = graphql(`
  query GetBillableRuns($month: Int!, $year: Int!) {
    runCountTimeSeries(timeOptions: { month: $month, year: $year }) {
      data {
        time
        value
      }
    }
  }
`);

export default function useGetUsageChartData({
  selectedPeriod,
  type,
}: {
  selectedPeriod: 'current' | 'previous';
  type: 'run' | 'step';
}) {
  const currentMonthIndex = new Date().getUTCMonth();

  const options = {
    previous: {
      month: currentMonthIndex === 0 ? 12 : currentMonthIndex,
      year: currentMonthIndex === 0 ? new Date().getUTCFullYear() - 1 : new Date().getUTCFullYear(),
    },
    current: {
      month: currentMonthIndex + 1,
      year: new Date().getUTCFullYear(),
    },
  };

  const query = type === 'step' ? GetBillableSteps : GetBillableRuns;

  const [{ data, fetching }] = useQuery<GetBillableStepsQuery | GetBillableRunsQuery>({
    query,
    variables: {
      month: options[selectedPeriod].month,
      year: options[selectedPeriod].year,
    },
  });

  let usageData: Array<TimeSeriesPoint> = [];

  if (type === 'step' && data && 'billableStepTimeSeries' in data) {
    usageData = data.billableStepTimeSeries[0]?.data || [];
  } else if (type === 'run' && data && 'runCountTimeSeries' in data) {
    usageData = data.runCountTimeSeries[0]?.data || [];
  }

  return {
    data: usageData,
    fetching,
  };
}
