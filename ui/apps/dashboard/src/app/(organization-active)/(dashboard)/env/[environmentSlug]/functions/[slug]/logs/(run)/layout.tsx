'use client';

import { useRouter } from 'next/navigation';
import { SlideOver } from '@inngest/components/SlideOver';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    slug: string;
  };
};

const GetFunctionRunTriggersDocument = graphql(`
  query GetFunctionRunTriggers($environmentID: ID!, $functionSlug: String!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        current {
          triggers {
            schedule
          }
        }
      }
    }
  }
`);

export default function RunLayout({ children, params }: RunLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);

  const router = useRouter();
  const environment = useEnvironment();
  const [{ data, fetching: fetchingRunTriggers }] = useQuery({
    query: GetFunctionRunTriggersDocument,
    variables: {
      environmentID: environment.id,
      functionSlug,
    },
  });

  const triggers = data?.environment.function?.current?.triggers;
  const hasCron =
    Array.isArray(triggers) && triggers.length > 0 && triggers.some((trigger) => trigger.schedule);

  if (fetchingRunTriggers) {
    return;
  }

  return (
    <SlideOver
      size={hasCron ? 'small' : 'large'}
      onClose={() =>
        router.push(
          pathCreator.functionRuns({
            envSlug: environment.slug,
            functionSlug,
          })
        )
      }
    >
      {children}
    </SlideOver>
  );
}
