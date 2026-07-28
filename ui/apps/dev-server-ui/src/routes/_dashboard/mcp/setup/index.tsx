import { createFileRoute } from '@tanstack/react-router';

import { MCPSetup } from '@inngest/components/MCP/Setup';
import { createDevServerURL } from '@/hooks/useFeatureFlags';

export const Route = createFileRoute('/_dashboard/mcp/setup/')({
  component: MCPComponent,
});

function MCPComponent() {
  const endpoint = createDevServerURL('/mcp');
  const absoluteEndpoint =
    typeof window === 'undefined'
      ? endpoint
      : new URL(endpoint, window.location.origin).toString();

  return <MCPSetup endpoint={absoluteEndpoint} isDevServer />;
}
