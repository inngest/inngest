import type { ComponentType } from 'react';
import { MCPIcon } from '@inngest/components/icons/sections/AI';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RunsIcon } from '@inngest/components/icons/sections/Runs';

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

export const setup: NavGroupConfig = {
  heading: 'Setup',
  items: [{ label: 'MCP', href: '/mcp', Icon: MCPIcon }],
};
