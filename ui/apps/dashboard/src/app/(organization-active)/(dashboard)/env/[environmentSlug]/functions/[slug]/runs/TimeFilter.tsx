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

export default function TimeFilter({ selectedDays, onDaysChange }: RelativeTimeFilterProps) {
  const [{ data }] = useQuery({
    query: GetBillingPlanDocument,
  });

  // Since "features" is a map, we can't be 100% sure that there's a log
  // retention value. So default to 7 days.
  let logRetention = 7;
  if (typeof data?.account.plan?.features.log_retention === 'number') {
    logRetention = data.account.plan.features.log_retention;
  }

  const options: Option[] = datesArray.map((date) => ({
    id: date,
    name: date === '1' ? `Last ${date} day` : `Last ${date} days`,
    disabled: parseInt(date) > logRetention,
  }));

  const selectedValue = options.find((option) => option.id === selectedDays.toString());

  /* TODO: better plan validation and toast when absolute time available */
  // If selected value is disabled, select 3 days instead
  if (selectedValue && selectedValue.disabled) {
    const newSelectedValue = options.find(
      (option) => !option.disabled && parseInt(option.id) === 3
    );
    if (newSelectedValue) {
      onDaysChange(newSelectedValue.id);
    }
  }

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
