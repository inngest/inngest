'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { ArrowPathIcon } from '@heroicons/react/20/solid';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Button from '@/components/Button';
import { getFragmentData, graphql, type FragmentType } from '@/gql';
import cn from '@/utils/cn';

const FunctionItemFragment = graphql(`
  fragment FunctionItem on Workflow {
    id
    slug
  }
`);

const RerunFunctionRunDocument = graphql(/* GraphQL */ `
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
  environmentSlug: string;
  environmentID: string;
  function_: FragmentType<typeof FunctionItemFragment>;
  functionRunID: string;
};

export default function RerunButton({
  environmentSlug,
  environmentID,
  functionRunID,
  ...props
}: RerunButtonProps) {
  const function_ = getFragmentData(FunctionItemFragment, props.function_);
  const [{ fetching: isMutating }, rerunFunctionRunMutation] =
    useMutation(RerunFunctionRunDocument);
  const router = useRouter();

  async function rerunFunction() {
    const response = await rerunFunctionRunMutation({
      environmentID,
      functionID: function_.id,
      functionRunID,
    });
    if (response.error) {
      toast.error('Failed to rerun function');
      return;
    }
    const newFunctionRunID = response.data?.retryWorkflowRun?.id as string;
    router.refresh();
    router.push(
      `/env/${environmentSlug}/functions/${function_.slug}/logs/${newFunctionRunID}` as Route
    );
  }

  return (
    <Button
      size="sm"
      context="dark"
      iconSide="right"
      variant="secondary"
      disabled={isMutating}
      onClick={() => rerunFunction()}
      icon={<ArrowPathIcon className={cn('h-3 w-3 text-sky-500', isMutating && 'animate-spin')} />}
    >
      Rerun
    </Button>
  );
}
