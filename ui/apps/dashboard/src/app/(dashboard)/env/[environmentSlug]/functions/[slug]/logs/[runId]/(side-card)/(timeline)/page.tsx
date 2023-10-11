import { notFound } from 'next/navigation';

import { graphql } from '@/gql';
import { RunHistoryType } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import RerunButton from './RerunButton';
import TimelineItem from './TimelineItem';
import TimelineStep from './TimelineStep';

export const dynamic = 'force-dynamic';

const GetFunctionRunTimelineDocument = graphql(/* GraphQL */ `
  query GetFunctionRunTimeline($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        ...FunctionItem
        run(id: $functionRunID) {
          canRerun

          timeline {
            stepName
            type: status
            output
            history {
              id
              type
              createdAt
              stepData {
                data
              }
            }
          }
        }
      }
    }
  }
`);

type FunctionTimelineProps = {
  params: {
    environmentSlug: string;
    slug: string;
    runId: string;
  };
};

export const runtime = 'nodejs';

export default async function FunctionTimeline({ params }: FunctionTimelineProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const response = await graphqlAPI.request(GetFunctionRunTimelineDocument, {
    environmentID: environment.id,
    functionSlug,
    functionRunID: params.runId,
  });

  const function_ = response.environment.function;

  if (!function_?.run.timeline) {
    notFound();
  }

  return (
    <div className="divide-y divide-slate-800 bg-slate-900 text-white">
      <div className="flex justify-between px-2 py-3">
        <h3>Function Timeline</h3>
        {function_.run.canRerun && (
          <RerunButton
            environmentSlug={environment.slug}
            environmentID={environment.id}
            function_={function_}
            functionRunID={params.runId}
          />
        )}
      </div>
      <div className="px-2 py-3">
        <ul role="list" className="pt-5">
          {function_.run.timeline.map((timelineItem, index) => {
            return timelineItem.type.startsWith('STEP_') ? (
              <TimelineStep
                name={timelineItem.stepName || 'Step'}
                isCompleted={timelineItem.type === RunHistoryType.StepCompleted}
                key={timelineItem.history![0]!.id}
              >
                {timelineItem.history?.map((stepTimelineItem) => {
                  return (
                    <TimelineItem
                      item={{
                        id: stepTimelineItem.id,
                        type: stepTimelineItem.type,
                        createdAt: stepTimelineItem.createdAt,
                        output: stepTimelineItem.stepData?.data,
                      }}
                      key={stepTimelineItem.id}
                    />
                  );
                })}
              </TimelineStep>
            ) : (
              <TimelineItem
                item={{
                  id: timelineItem.history?.[0]?.id,
                  type: timelineItem.type,
                  createdAt: timelineItem.history?.[0]?.createdAt,
                  output:
                    timelineItem.output && timelineItem.output !== 'null'
                      ? timelineItem.output
                      : undefined,
                }}
                isFirst={index === 0}
                isLast={index === (function_.run.timeline?.length ?? 0) - 1}
                key={index}
              />
            );
          })}
        </ul>
      </div>
    </div>
  );
}
