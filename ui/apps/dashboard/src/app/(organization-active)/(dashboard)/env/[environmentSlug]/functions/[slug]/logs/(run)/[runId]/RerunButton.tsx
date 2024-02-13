'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { ArrowPathIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import type { Environment } from '@inngest/components/types/environment';
import type { Function } from '@inngest/components/types/function';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import cn from '@/utils/cn';

export const RerunFunctionRunDocument = graphql(/* GraphQL */ `
  mutation RerunFunctionRun($environmentID: ID!, $functionID: ID!, $functionRunID: ULID!) {
    retryWorkflowRun(
      input: { workspaceID: $environmentID, workflowID: $functionID }
      workflowRunID: $functionRunID
    ) {
      id
    }
  }
`);

type RerunButtonProps = {
  environment: Pick<Environment, 'id' | 'slug'>;
  func: Pick<Function, 'id' | 'slug'>;
  functionRunID: string;
};

export default function RerunButton({ environment, functionRunID, func }: RerunButtonProps) {
  const [{ fetching: isMutating }, rerunFunctionRunMutation] =
    useMutation(RerunFunctionRunDocument);
  const router = useRouter();

  async function rerunFunction() {
    const response = await rerunFunctionRunMutation({
      environmentID: environment.id,
      functionID: func.id,
      functionRunID,
    });
    if (response.error) {
      toast.error('Failed to rerun function');
      return;
    } else {
      toast.success('Successfully rerun. Loading new run...');
    }
    const newFunctionRunID = response.data?.retryWorkflowRun?.id as string;
    router.refresh();
    router.push(
      `/env/${environment.slug}/functions/${encodeURIComponent(
        func.slug
      )}/logs/${newFunctionRunID}` as Route
    );
  }

  return (
    <Button
      size="small"
      iconSide="right"
      loading={isMutating}
      btnAction={() => rerunFunction()}
      icon={<ArrowPathIcon className={cn(' text-sky-500', isMutating && 'animate-spin')} />}
      label={isMutating ? 'Running...' : 'Rerun'}
    />
  );
}
