'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { SlideOver } from '@inngest/components/SlideOver';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
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
  const router = useRouter();

  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const [{ data, fetching: fetchingRunTriggers }] = useQuery({
    query: GetFunctionRunTriggersDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug: params.slug,
    },
    pause: !environment?.id,
  });

  const triggers = data?.environment?.function?.current?.triggers;
  const hasCron =
    Array.isArray(triggers) && triggers.length > 0 && triggers.some((trigger) => trigger.schedule);

  if (isFetchingEnvironment || fetchingRunTriggers) {
    return;
  }

  return (
    <SlideOver
      size={hasCron ? 'small' : 'large'}
      onClose={() =>
        router.push(
          `/env/${params.environmentSlug}/functions/${encodeURIComponent(
            params.slug
          )}/logs` as Route
        )
      }
    >
      {children}
    </SlideOver>
  );
}
