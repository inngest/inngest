import { Alert } from '@inngest/components/Alert';

import { SdkMode, SecretCheck, type AppCheckResult } from '@/gql/graphql';
import { isAppInfoMissingData } from './utils';

type Props = {
  appInfo: AppCheckResult;
};

export function Checks({ appInfo }: Props) {
  if (!appInfo.isReachable) {
    return (
      <Alert className="mb-4" severity="error">
        App is not reachable
      </Alert>
    );
  }

  const issues = [];
  for (const [id, check] of Object.entries(checks)) {
    const result = check(appInfo);
    if (result) {
      issues.push({ id, result });
    }
  }

  if (issues.length === 0) {
    if (appInfo.error) {
      return (
        <Alert className="mb-4" severity="error">
          Error: {appInfo.error}
        </Alert>
      );
    }

    if (appInfo.signingKeyStatus === SecretCheck.Unknown) {
      return (
        <Alert className="mb-4" severity="warning">
          Unable to get full SDK info. Your SDK may need to be updated.
        </Alert>
      );
    }

    if (isAppInfoMissingData(appInfo)) {
      return (
        <Alert className="mb-4" severity="warning">
          No issues found. However, not all checks were performed since data is missing. Updating
          your SDK should resolve this since newer SDK versions report more data.
        </Alert>
      );
    }

    return (
      <Alert className="mb-4" severity="success">
        No issues found
      </Alert>
    );
  }

  return (
    <div className="mb-4 flex flex-col gap-2">
      {issues
        .sort(({ result }) => {
          // Sort by severity
          if (result.severity === 'critical') {
            return -1;
          }
          if (result.severity === 'error') {
            return 0;
          }
          return 1;
        })
        .map(({ id, result }) => {
          let severity: React.ComponentProps<typeof Alert>['severity'];
          if (result.severity === 'critical') {
            severity = 'error';
          } else {
            severity = result.severity;
          }

          return (
            <Alert key={id} severity={severity}>
              {result.message}
            </Alert>
          );
        })}
    </div>
  );
}

type CheckResult = {
  message: string;
  severity: 'critical' | 'error' | 'warning';
};

const checks: Record<string, (appInfo: AppCheckResult) => CheckResult | undefined> = {
  apiOrigin: (appInfo) => {
    if (!appInfo.apiOrigin?.value) {
      return;
    }

    if (
      !['https://api.inngest.com', 'https://api.inngest.com/'].includes(appInfo.apiOrigin.value)
    ) {
      return {
        message: `Non-standard API origin: ${appInfo.apiOrigin.value}`,
        severity: 'error',
      };
    }
  },
  authentication: (appInfo) => {
    if (appInfo.authenticationSucceeded?.value === false) {
      return {
        message: 'Authentication failed. Your SDK may be using the wrong signing key',
        severity: 'error',
      };
    }
  },
  eventAPIOrigin: (appInfo) => {
    if (!appInfo.eventAPIOrigin?.value) {
      return;
    }

    if (!['https://inn.gs/', 'https://inn.gs/'].includes(appInfo.eventAPIOrigin.value)) {
      return {
        message: `Non-standard event API origin: ${appInfo.eventAPIOrigin.value}`,
        severity: 'error',
      };
    }
  },
  eventKey: (appInfo) => {
    if (appInfo.eventKeyStatus === SecretCheck.Incorrect) {
      return {
        message: 'Event key is incorrect',
        severity: 'error',
      };
    }

    if (appInfo.eventKeyStatus === SecretCheck.Missing) {
      return {
        message: 'No event key',
        severity: 'warning',
      };
    }
  },
  isSDK: (appInfo) => {
    if (appInfo.respStatusCode && appInfo.respStatusCode !== 200) {
      return;
    }

    if (!appInfo.isSDK) {
      return {
        message: 'Response did not come from an Inngest SDK',
        severity: 'error',
      };
    }
  },
  mode: (appInfo) => {
    if (!appInfo.mode) {
      return;
    }

    if (appInfo.mode !== SdkMode.Cloud) {
      return {
        message: `Not in Cloud mode`,
        severity: 'error',
      };
    }
  },
  signingKey: (appInfo) => {
    if (appInfo.signingKeyStatus === SecretCheck.Incorrect) {
      return {
        message: 'Signing key is incorrect',
        severity: 'error',
      };
    }

    if (appInfo.signingKeyStatus === SecretCheck.Missing) {
      return {
        message: 'No signing key',
        severity: 'error',
      };
    }
  },
};
