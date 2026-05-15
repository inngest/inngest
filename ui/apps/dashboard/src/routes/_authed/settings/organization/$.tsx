import { useEffect, useState, type FormEvent } from 'react';
import {
  OrganizationProfile,
  useOrganization,
} from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { createFileRoute, useLocation } from '@tanstack/react-router';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import { graphql } from '@/gql';

const SecurityEmailSettingsDocument = graphql(`
  query SecurityEmailSettings {
    account {
      securityEmail
    }
  }
`);

const UpdateSecurityEmailDocument = graphql(`
  mutation UpdateSecurityEmail($input: UpdateAccount!) {
    account: updateAccount(input: $input) {
      securityEmail
    }
  }
`);

export const Route = createFileRoute('/_authed/settings/organization/$')({
  component: OrganizationSettingsPage,
});

function OrganizationSettingsPage() {
  const location = useLocation();
  const isBaseOrganizationPage =
    location.pathname === '/settings/organization' ||
    location.pathname === '/settings/organization/';

  return (
    <div className="flex w-full flex-col justify-start">
      <OrganizationProfile
        key={location.pathname}
        routing="path"
        path="/settings/organization"
        appearance={{
          layout: {
            logoPlacement: 'none',
          },
          elements: {
            navbar: 'hidden',
            scrollBox: 'bg-canvasBase shadow-none',
            pageScrollBox: 'pt-6 px-2 w-full',
          },
        }}
      />
      {isBaseOrganizationPage && <SecurityEmailSettings />}
    </div>
  );
}

function SecurityEmailSettings() {
  const [{ data, error, fetching }, refetch] = useQuery({
    query: SecurityEmailSettingsDocument,
  });
  const [{ fetching: isSaving }, updateSecurityEmail] = useMutation(
    UpdateSecurityEmailDocument,
  );
  const { isLoaded: orgLoaded, membership } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';
  const currentSecurityEmail = data?.account.securityEmail ?? '';
  const [securityEmail, setSecurityEmail] = useState(currentSecurityEmail);
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    setSecurityEmail(currentSecurityEmail);
  }, [currentSecurityEmail]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!isAdmin) return;

    const trimmedSecurityEmail = securityEmail.trim();
    setSaveError(null);

    const res = await updateSecurityEmail(
      { input: { securityEmail: trimmedSecurityEmail } },
      { additionalTypenames: ['Account'] },
    );
    if (res.error) {
      setSaveError(res.error.message.replace('[GraphQL] ', ''));
      return;
    }

    setSecurityEmail(res.data?.account.securityEmail ?? '');
    refetch({ requestPolicy: 'network-only' });
    toast.success('Security email updated');
  }

  const canEdit = orgLoaded && isAdmin;
  const trimmedSecurityEmail = securityEmail.trim();
  const isDirty = trimmedSecurityEmail !== currentSecurityEmail;

  return (
    <div className="mx-auto mb-8 w-fit max-w-[1200px] px-6">
      <div className="w-fit bg-canvasBase shadow-none md:min-w-[880px]">
        <div className="px-2 pt-6">
          <section className="border-subtle bg-canvasBase w-full rounded-md border p-6 md:w-[784px]">
            <div className="mb-6 flex flex-col gap-1">
              <h2 className="text-muted text-lg">Security email</h2>
              <p className="text-muted text-sm">
                This account-level email receives security notifications for
                your organization.
              </p>
            </div>
            <form className="flex flex-col gap-4" onSubmit={submit}>
              {error && <Alert severity="error">{error.message}</Alert>}
              {saveError && <Alert severity="error">{saveError}</Alert>}
              <Input
                label="Email address"
                name="security-email"
                type="email"
                value={securityEmail}
                placeholder="security@example.com"
                onChange={(event) => setSecurityEmail(event.target.value)}
                readOnly={!canEdit || fetching || isSaving}
              />
              {!canEdit && orgLoaded && (
                <p className="text-subtle text-sm">
                  Only organization admins can update the security email.
                </p>
              )}
              <div className="flex justify-end">
                <Button
                  appearance="outlined"
                  kind="primary"
                  label="Save"
                  type="submit"
                  loading={isSaving}
                  disabled={!canEdit || fetching || isSaving || !isDirty}
                />
              </div>
            </form>
          </section>
        </div>
      </div>
    </div>
  );
}
