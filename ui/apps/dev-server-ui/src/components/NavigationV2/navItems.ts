import type { ComponentType } from 'react';
import { MCPIcon } from '@inngest/components/icons/sections/AI';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs';
import { ExperimentsIcon } from '@inngest/components/icons/sections/Experiments';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { InsightsIcon } from '@inngest/components/icons/sections/Insights';
import { RunsIcon } from '@inngest/components/icons/sections/Runs';
import { ScoresIcon } from '@inngest/components/icons/sections/Scores';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';

export type NavItemConfig = {
  label: string;
  // Absolute route — the dev server has no environment prefix.
  href: string;
  Icon: ComponentType<{ className?: string }>;
  exact?: boolean;
};

export type NavGroupConfig = {
  heading: string;
  items: NavItemConfig[];
  // Renders a "Beta" pill next to the group heading.
  beta?: boolean;
};

export const workflow: NavGroupConfig = {
  heading: 'Workflow',
  items: [
    { label: 'Apps', href: '/apps', Icon: AppsIcon },
    { label: 'Functions', href: '/functions', Icon: FunctionsIcon },
    { label: 'Runs', href: '/runs', Icon: RunsIcon },
    { label: 'Events', href: '/events', Icon: EventLogsIcon },
  ],
};

// Only shown when the "duckdb-insights" feature flag is enabled (devserver
// started with --duckdb) -- see Navigation.tsx.
export const insightsNavItem: NavItemConfig = {
  label: 'Insights',
  href: '/insights',
  Icon: InsightsIcon,
};

// Sessions reads from the same DuckDB dual-write store as Insights
// (pkg/duckdb/dashboards), so it's gated behind the same "duckdb-insights"
// flag -- see Navigation.tsx.
export const sessionsNavItem: NavItemConfig = {
  label: 'Sessions',
  href: '/sessions',
  Icon: SessionsIcon,
};

// Item order matches the cloud dashboard's AI section.
export const ai: NavGroupConfig = {
  heading: 'AI',
  beta: true,
  items: [
    { label: 'Experiments', href: '/ai/experiments', Icon: ExperimentsIcon },
    { label: 'Scores', href: '/ai/scores', Icon: ScoresIcon },
  ],
};

export const setup: NavGroupConfig = {
  heading: 'Setup',
  items: [{ label: 'MCP', href: '/mcp/setup', Icon: MCPIcon }],
};
