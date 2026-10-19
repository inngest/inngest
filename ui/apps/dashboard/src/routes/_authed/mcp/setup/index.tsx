import { MCPSetup } from '@inngest/components/MCP/Setup';
import { Header } from '@inngest/components/Header/Header';
import { Button } from '@inngest/components/Button';
import { createFileRoute } from '@tanstack/react-router';

import { validateRunAISetupSearch } from '@/components/Runs/AIAssist/setup';
import { useRunAITracking } from '@/components/Runs/AIAssist/tracking';

const MCPSetupPage = () => {
  const { client, returnTo } = Route.useSearch();
  const { track } = useRunAITracking(
    returnTo?.includes('/functions/') ? 'fn' : 'env',
    returnTo?.split('?')[1] ?? '',
  );
  return (
    <div className="flex h-full flex-col">
      <Header backNav breadcrumb={[{ text: 'MCP setup' }]} />
      <div
        className="no-scrollbar min-h-0 flex-1 overflow-y-auto"
        onClick={(event) => {
          if (!returnTo || !(event.target instanceof Element)) return;
          const anchor = event.target.closest('a');
          if (anchor?.href.startsWith('https://www.inngest.com/docs/')) {
            track({
              action: 'guide_clicked',
              guide: anchor.href.includes('agent-skills')
                ? 'more_ai_resources'
                : anchor.href.includes('troubleshooting')
                  ? 'troubleshooting'
                  : 'connection',
            });
          }
        }}
      >
        {returnTo && (
          <div className="mx-auto max-w-5xl px-8 pt-6">
            <Button
              href={returnTo}
              label="Back to Runs"
              kind="secondary"
              appearance="outlined"
            />
            <p className="text-muted mt-3 text-sm">
              Connect in your AI tool, choose read access for the intended
              account and environment, then return to Runs to copy your prompt.
            </p>
            {client === 'claude' && (
              <p className="text-muted mt-2 text-sm">
                The official Inngest plugin includes Cloud MCP.{' '}
                <a
                  className="text-link hover:underline"
                  href="https://www.inngest.com/docs/ai-dev-tools/agent-skills#claude-code-plugin"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  View plugin instructions
                </a>
                , or use the direct MCP setup below. Use one connection method
                to avoid duplicates.
              </p>
            )}
          </div>
        )}
        <MCPSetup
          initialClient={client}
          apiKeysHref="/settings/api-keys"
          bearerTokenEnvVar="INNGEST_API_KEY"
          endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
          operationsEndpoint={new URL(
            '/v2/operations',
            import.meta.env.VITE_API_URL,
          ).toString()}
        />
      </div>
    </div>
  );
};

export const Route = createFileRoute('/_authed/mcp/setup/')({
  validateSearch: validateRunAISetupSearch,
  component: MCPSetupPage,
});
