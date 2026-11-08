import type { ComponentType } from 'react';
import type { TabsProps } from '@inngest/components/CodeBlock/CommandBlock';
import { IconClaude } from '@inngest/components/icons/ai/Claude';
import { IconCursor } from '@inngest/components/icons/ai/Cursor';
import { IconOpenAI } from '@inngest/components/icons/ai/OpenAI';

export type MCPClient = {
  id: 'claude' | 'codex' | 'cursor';
  name: string;
  Icon: ComponentType<{ className?: string; size?: number }>;
  snippet: TabsProps;
};

export const createMCPClients = (
  endpoint: string,
  isDevServer: boolean,
  bearerTokenEnvVar?: string
): MCPClient[] => {
  const serverName = isDevServer ? 'inngest-dev' : 'inngest-cloud';
  const authorizationHeader = bearerTokenEnvVar
    ? ` --header "Authorization: Bearer $${bearerTokenEnvVar}"`
    : '';
  const bearerTokenOption = bearerTokenEnvVar ? ` --bearer-token-env-var ${bearerTokenEnvVar}` : '';

  return [
    {
      id: 'claude',
      name: 'Claude Code',
      Icon: IconClaude,
      snippet: {
        title: 'Terminal command',
        language: 'shell',
        content: `claude mcp add --transport http ${serverName} ${endpoint}${authorizationHeader}`,
      },
    },
    {
      id: 'codex',
      name: 'Codex CLI',
      Icon: IconOpenAI,
      snippet: {
        title: 'Terminal command',
        language: 'shell',
        content: `codex mcp add ${serverName} --url ${endpoint}${bearerTokenOption}`,
      },
    },
    {
      id: 'cursor',
      name: 'Cursor',
      Icon: IconCursor,
      snippet: {
        title: 'mcp.json',
        language: 'json',
        content: JSON.stringify(
          {
            mcpServers: {
              [serverName]: {
                url: endpoint,
                ...(!isDevServer &&
                  !bearerTokenEnvVar && {
                    auth: {
                      CLIENT_ID: 'inngest-mcp-static',
                    },
                  }),
                ...(bearerTokenEnvVar && {
                  headers: {
                    Authorization: `Bearer \${env:${bearerTokenEnvVar}}`,
                  },
                }),
              },
            },
          },
          null,
          2
        ),
      },
    },
  ];
};
