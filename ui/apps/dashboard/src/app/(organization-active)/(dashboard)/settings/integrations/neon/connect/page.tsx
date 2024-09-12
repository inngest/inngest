'use client';

import ConnectPage from '@inngest/components/PostgresIntegrations/ConnectPage';
import { neonConnectContent } from '@inngest/components/PostgresIntegrations/neonContent';

export default function NeonConnect() {
  return <ConnectPage content={neonConnectContent} onStartInstallation={() => {}} />;
}
