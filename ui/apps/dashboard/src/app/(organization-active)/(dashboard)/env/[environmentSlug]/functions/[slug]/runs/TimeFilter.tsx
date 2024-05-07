import RelativeTimeFilter from '@inngest/components/Filter/RelativeTimeFilter';
import { type Option } from '@inngest/components/Select/Select';
import { useQuery } from 'urql';

import { graphql } from '@/gql';

const GetBillingPlanDocument = graphql(`
  query GetBillingPlan {
    account {
      plan {
        id
        name
        features
      }
    }

    plans {
      name
      features
    }
  }
`);

type RelativeTimeFilterProps = {
  selectedDays: string;
  onDaysChange: (value: string) => void;
};

const datesArray = ['1', '3', '7', '14', '30'];
const options: Option[] = datesArray.map((date) => ({
  id: date,
  name: `Last ${date} days`,
  disabled: false,
}));

export default function TimeFilter({ selectedDays, onDaysChange }: RelativeTimeFilterProps) {
  const [{ data }] = useQuery({
    query: GetBillingPlanDocument,
  });

  const selectedValue = options.find((option) => option.id === selectedDays.toString());
  return (
    <RelativeTimeFilter
      options={options}
      selectedDays={selectedValue}
      onDaysChange={(value: Option) => {
        const numericId = parseInt(value.id);
        if (!isNaN(numericId)) {
          onDaysChange(value.id);
        }
      }}
    />
  );
}
