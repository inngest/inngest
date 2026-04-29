import { useEffect, useState, type FormEvent } from 'react';
import { useOrganization, UserProfile } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { createFileRoute, redirect } from '@tanstack/react-router';
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

export const Route = createFileRoute('/_authed/settings/user/$')({
  gcTime: 0,
  ssr: false,
  component: UserSettingsPage,
  beforeLoad: ({ location }) => {
    if (
      location.pathname === '/settings/user' ||
      location.pathname === '/settings/user/'
    ) {
      throw redirect({ to: '/settings/user/$', params: { _splat: 'profile' } });
    }
  },
});

function UserSettingsPage() {
  const { _splat } = Route.useParams();

  return (
    <div className="flex flex-col justify-start">
      {_splat === 'profile' && (
        <UserProfile
          routing="path"
          path="/settings/user/profile"
          appearance={{
            layout: {
              logoPlacement: 'none',
            },
            elements: {
              navbar: 'hidden',
              scrollBox: 'bg-canvasBase shadow-none',
              pageScrollBox: 'pt-6 px-2',
            },
          }}
        >
          <UserProfile.Page label="security" />
        </UserProfile>
      )}
      {_splat === 'security' && (
        <>
          <UserProfile
            routing="path"
            path="/settings/user/security"
            appearance={{
              layout: {
                logoPlacement: 'none',
              },
              elements: {
                navbar: 'hidden',
                scrollBox: 'bg-canvasBase shadow-none',
                pageScrollBox: 'pt-6 px-2',
              },
            }}
          >
            <UserProfile.Page label="account" />
          </UserProfile>
          <SecurityEmailSettings />
        </>
      )}
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
          <section className="border-subtle bg-canvasBase w-full rounded-md border p-6 md:w-[770px]">
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
