import { MCPSetup } from '@inngest/components/MCP/Setup';
import { Header } from '@inngest/components/Header/Header';
import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute } from '@tanstack/react-router';

const MCPSetupPage = () => {
  const { getToken } = useAuth();

  return (
    <div className="h-full flex-col">
      <Header backNav breadcrumb={[{ text: 'MCP setup' }]} />
      <div className="no-scrollbar h-full overflow-y-scroll">
        <MCPSetup
          bearerTokenEnvVar="INNGEST_API_KEY"
          endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
          getAccessToken={getToken}
        />
      </div>
    </div>
  );
};

export const Route = createFileRoute('/_authed/mcp/setup/')({
  component: MCPSetupPage,
});
