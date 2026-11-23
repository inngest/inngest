import { useQuery } from 'urql';
import { useEnvironment } from '@/components/Environments/environment-context';
import { ComputeVerificationDocument } from '../ComputeVerification/operations';
import { SandboxesLayout } from './SandboxesLayout';
import { SandboxesList } from './SandboxesList';
import SandboxesEmptyState from './SandboxesEmptyState';
import type { SandboxesVerificationState } from './SandboxesVerificationCallout';

export function SandboxesAccess() {
  const environment = useEnvironment();
  const [result, refresh] = useQuery({
    query: ComputeVerificationDocument,
    requestPolicy: 'network-only',
  });
  const account = result.data?.account;
  const summary = account?.computeVerification;
  let state: SandboxesVerificationState = { status: 'loading' };
  if (result.error || (!result.fetching && !summary))
    state = { status: 'error' };
  else if (summary) {
    switch (summary.status) {
      case 'REQUIRED': {
        const completed = summary.checks.filter((check) => check.passed).length;
        state = summary.canVerify
          ? {
              status: 'required',
              canVerify: true,
              started: completed > 0,
              progress: { completed, total: summary.checks.length },
            }
          : { status: 'required', canVerify: false };
        break;
      }
      case 'SUSPENDED':
        state = { status: 'blocked' };
        break;
      case 'VERIFIED':
      case 'EXEMPT':
        return (
          <SandboxesLayout key={environment.id}>
            <SandboxesList />
          </SandboxesLayout>
        );
      case 'NOT_ELIGIBLE':
        if (account?.ents.sandboxes.enabled)
          return (
            <SandboxesLayout key={environment.id}>
              <SandboxesList />
            </SandboxesLayout>
          );
        return <SandboxesEmptyState />;
    }
  }
  return (
    <SandboxesEmptyState
      requiresUpgrade={false}
      verification={{
        state,
        onRetry: () => refresh({ requestPolicy: 'network-only' }),
        onVerify: () => {
          if (!account || summary?.status !== 'REQUIRED' || !summary.canVerify)
            return;
          const query = new URLSearchParams({
            account_id: account.id,
            source: 'dashboard',
            return_to: `/env/${environment.slug}/sandboxes`,
          });
          window.location.assign(`/verify?${query}`);
        },
      }}
    />
  );
}
