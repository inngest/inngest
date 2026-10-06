import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

const sandboxAccessDocument = graphql(`
  query SandboxAccess {
    account {
      id
      entitlements: ents {
        sandboxes {
          enabled
        }
      }
    }
  }
`);

// Tells CLI users before they approve a login that Cloud sandboxes from the
// dev server need a paid plan. Renders nothing while loading, on error, or
// when the account already has access.
export function SandboxUpgradeNotice({ refSource }: { refSource: string }) {
  const [{ data }] = useQuery({ query: sandboxAccessDocument });
  if (data?.account.entitlements.sandboxes.enabled !== false) return null;

  return (
    <Alert
      severity="info"
      button={
        <Button
          kind="primary"
          appearance="outlined"
          label="Upgrade plan"
          href={pathCreator.billing({ tab: 'plans', ref: refSource })}
          target="_blank"
          rel="noopener noreferrer"
        />
      }
    >
      Cloud sandboxes require a paid plan. You can still log in, but sandbox
      requests from the dev server will be rejected until you upgrade.
    </Alert>
  );
}
