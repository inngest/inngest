import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button/NewButton';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, runQuery } = useInsightsStateMachineContext();

  return (
    <Banner
      cta={
        <Button
          appearance="ghost"
          kind="danger"
          label="Retry"
          onClick={() => {
            runQuery();
          }}
        />
      }
      severity="error"
    >
      {error ? pruneGraphQLError(error) : FALLBACK_ERROR}
    </Banner>
  );
}

function pruneGraphQLError(error: Error) {
  return error.message.replace(/^\[GraphQL\] /, '');
}
