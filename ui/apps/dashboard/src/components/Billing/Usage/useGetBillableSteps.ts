import { useQuery } from 'urql';

import { graphql } from '@/gql';

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

export default function useGetBillableSteps({
  selectedPeriod,
}: {
  selectedPeriod: 'current' | 'previous';
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

  const [{ data, fetching }] = useQuery({
    query: GetBillableSteps,
    variables: {
      month: options[selectedPeriod].month,
      year: options[selectedPeriod].year,
    },
  });

  return {
    data: data?.billableStepTimeSeries[0]?.data || [],
    fetching,
  };
}
