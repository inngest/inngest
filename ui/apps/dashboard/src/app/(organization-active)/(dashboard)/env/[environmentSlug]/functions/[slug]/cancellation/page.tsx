'use client';

import React from 'react';
import { useMutation } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { CancellationList } from './CancellationList';
import { CreateCancellationButton } from './CreateCancellationButton';

const CreateCancellationDocument = graphql(`
  mutation CreateCancellation($input: CreateCancellationInput!) {
    createCancellation(input: $input) {
      id
    }
  }
`);

const GetFunctionCancellationPageDocument = graphql(`
  query GetFunctionCancellationPage($envID: ID!, $fnSlug: String!) {
    environment: workspace(id: $envID) {
      function: workflowBySlug(slug: $fnSlug) {
        id
      }
    }
  }
`);

type Props = {
  params: {
    slug: string;
  };
};

export default function Page({ params }: Props) {
  const fnSlug = decodeURIComponent(params.slug);
  const [, createCancellation] = useMutation(CreateCancellationDocument);
  const envID = useEnvironment().id;

  const fnRes = useGraphQLQuery({
    query: GetFunctionCancellationPageDocument,
    variables: { envID, fnSlug },
  });
  if (fnRes.error) {
    throw fnRes.error;
  }
  if (fnRes.isLoading) {
    return null;
  }

  const fnID = fnRes.data.environment.function?.id;
  if (!fnID) {
    throw new Error('Function not found');
  }

  return (
    <>
      <div className="flex items-center justify-end border-b border-slate-300 px-5 py-2">
        <CreateCancellationButton
          onSubmit={async (data) => {
            const res = await createCancellation({
              input: {
                envID,
                expression: data.expression,
                functionID: fnID,
                name: data.name,
                queuedAtMax: data.queuedAtMax.toISOString(),
                queuedAtMin: data.queuedAtMin?.toISOString(),
              },
            });
            if (res.error) {
              // Throw error so that the modal can catch and display it
              throw res.error;
            }
          }}
        />
      </div>

      <div className="overflow-y-auto">
        <CancellationList envID={envID} fnID={fnID} />
      </div>
    </>
  );
}
