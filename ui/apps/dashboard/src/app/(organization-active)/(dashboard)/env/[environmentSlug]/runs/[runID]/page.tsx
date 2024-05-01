'use client';

import { RunDetails } from '@inngest/components/RunDetailsV2';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { getFragmentData, graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const TraceDetailsFragment = graphql(`
  fragment TraceDetails on RunTraceSpan {
    name
    status
    attempts
    queuedAt
    startedAt
    endedAt
    isRoot
    outputID
    spanID
    stepOp
    stepInfo {
      __typename
      ... on InvokeStepInfo {
        triggeringEventID
        functionID
        timeout
        returnEventID
        runID
        timedOut
      }
      ... on SleepStepInfo {
        sleepUntil
      }
      ... on WaitForEventStepInfo {
        eventName
        expression
        timeout
        foundEventID
        timedOut
      }
    }
  }
`);

const QueryDocument = graphql(`
  query GetRunTrace($envID: ID!, $runID: String!) {
    workspace(id: $envID) {
      run(runID: $runID) {
        trace {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
            }
          }
        }
      }
    }
  }
`);

type Props = {
  params: {
    runID: string;
  };
};

export default function Page({ params }: Props) {
  const env = useEnvironment();

  const res = useGraphQLQuery({
    //   const [res] = useQuery({
    query: QueryDocument,
    variables: {
      envID: env.id,
      runID: params.runID,
    },
  });
  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return <Loading />;
  }
  if (!res.data.workspace.run) {
    throw new Error('missing run');
  }

  async function getOutput() {
    return null;
  }

  const { trace } = res.data.workspace.run;
  if (!trace) {
    throw new Error('missing trace');
  }

  return (
    <RunDetails
      app={{ name: 'my-app' }}
      fn={{ name: 'my-fn' }}
      getOutput={getOutput}
      run={{
        id: params.runID,
        output: null,
        trace: getFragmentData(TraceDetailsFragment, trace),
      }}
    />
  );
}

function Loading() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="flex flex-col items-center justify-center gap-2">
        <LoadingIcon />
        <div>Loading</div>
      </div>
    </div>
  );
}
