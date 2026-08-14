import { MCPSetup } from '@inngest/components/MCP/Setup';
import { Header } from '@inngest/components/Header/Header';
import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute } from '@tanstack/react-router';

const MCPSetupPage = () => {
  const { getToken, isLoaded } = useAuth();

  return (
    <div className="flex h-full flex-col">
      <Header backNav breadcrumb={[{ text: 'MCP setup' }]} />
      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto">
        <MCPSetup
          apiKeysHref="/settings/api-keys"
          bearerTokenEnvVar="INNGEST_API_KEY"
          endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
          getAccessToken={getToken}
          isAuthReady={isLoaded}
        />
      </div>
    </div>
  );
};

export const Route = createFileRoute('/_authed/mcp/setup/')({
  component: MCPSetupPage,
});
