import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiArrowLeftLine } from '@remixicon/react';
import {
  createFileRoute,
  getRouteApi,
  useNavigate,
} from '@tanstack/react-router';

import { CreateAPIKeyForm } from '@/components/APIKeys/CreateAPIKeyForm';
import { canManageAPIKeys } from '@/components/APIKeys/permissions';
import LoadingIcon from '@/components/Icons/LoadingIcon';

type CreateAPIKeySearch = {
  type?: 'user';
};

export const Route = createFileRoute('/_authed/settings/create-api-key/')({
  component: CreateAPIKeyPage,
  validateSearch: (search: Record<string, unknown>): CreateAPIKeySearch => {
    if (search.type === 'user') {
      return { type: 'user' };
    }
    return {};
  },
});

const authedRoute = getRouteApi('/_authed');

function CreateAPIKeyPage() {
  const navigate = useNavigate();
  const search = Route.useSearch();
  const { profile } = authedRoute.useLoaderData();
  const { membership, isLoaded: orgLoaded } = useOrganization();
  const canManage = canManageAPIKeys({
    marketplace: profile.marketplace,
    organizationRole: membership?.role,
  });
  const createUserAPIKey = search.type === 'user';
  let pageTitle = 'Create Service API key';
  if (createUserAPIKey) {
    pageTitle = 'Create User API key';
  }

  const backToList = () => navigate({ to: '/settings/api-keys' });

  if (!canManage && !orgLoaded) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  let pageContent = (
    <Alert severity="error">
      Only organization admins can manage API keys.
    </Alert>
  );
  if (canManage) {
    pageContent = (
      <CreateAPIKeyForm
        createUserAPIKey={createUserAPIKey}
        canCreateServiceKeys={canManage}
        onCancel={backToList}
        onDone={backToList}
      />
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-8">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-basis text-2xl">{pageTitle}</h1>
        </div>
        <Button
          appearance="outlined"
          kind="secondary"
          icon={<RiArrowLeftLine />}
          iconSide="left"
          label="Back"
          onClick={backToList}
        />
      </div>

      {pageContent}
    </div>
  );
}
