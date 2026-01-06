import useManagePageTerminology from '@/components/Manage/useManagePageTerminology';
import { EnvironmentType } from '@/utils/environments';
import { createFileRoute } from '@tanstack/react-router';
import { Route as OrgActiveRoute } from '@/routes/_authed';
import { Alert } from '@inngest/components/Alert';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/manage/$ingestKeys/',
)({
  component: KeysComponent,
});

function KeysComponent() {
  const currentContent = useManagePageTerminology();
  const { env } = OrgActiveRoute?.useLoaderData() ?? {};

  const shouldShowAlert =
    currentContent?.param === 'keys' &&
    env?.type === EnvironmentType.BranchParent;

  return (
    <div className="flex h-full w-full flex-col">
      {shouldShowAlert && (
        <Alert
          className="flex items-center rounded-none text-sm"
          severity="info"
        >
          Event keys are shared for all branch environments
        </Alert>
      )}
      <div className="flex flex-1 items-center justify-center">
        <h2 className="text-subtle text-sm font-semibold">
          {'Select a ' + currentContent?.type + ' on the left.'}
        </h2>
      </div>
    </div>
  );
}
