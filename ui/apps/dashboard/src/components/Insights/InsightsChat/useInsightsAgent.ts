import {
  useAgents,
  type AgentKitEvent,
  type UseAgentsConfig,
  type UseAgentsReturn,
} from '@inngest/use-agents';

import type { ToolManifest } from '@/app/api/inngest/functions/agents/types';

export type ClientState = {
  sqlQuery: string;
  eventTypes: string[];
  schemas: Record<string, unknown> | null;
  currentQuery: string;
  tabTitle: string;
  mode: 'insights_sql_playground';
  timestamp: number;
};

export type InsightsAgentConfig = { tools: ToolManifest; state: ClientState };

export type InsightsAgentEvent = AgentKitEvent<ToolManifest>;

export function useInsightsAgent(
  config: UseAgentsConfig<ToolManifest, ClientState>
): UseAgentsReturn<ToolManifest, ClientState> {
  return useAgents<{ tools: ToolManifest; state: ClientState }>(config);
}
