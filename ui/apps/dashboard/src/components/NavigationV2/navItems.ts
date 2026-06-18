import type { ComponentType } from 'react';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { ExperimentsIcon } from '@inngest/components/icons/sections/Experiments';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { InsightsIcon } from '@inngest/components/icons/sections/Insights';
import { MetricsIcon } from '@inngest/components/icons/sections/Metrics';
import { OverviewIcon } from '@inngest/components/icons/sections/Overview';
import { RunsIcon } from '@inngest/components/icons/sections/Runs';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import { WebhooksIcon } from '@inngest/components/icons/sections/Webhooks';

export type NavItemConfig = {
  label: string;
  // Relative path passed to getNavRoute (env slug is prepended there).
  route: string;
  Icon: ComponentType<{ className?: string }>;
  beta?: boolean;
  exact?: boolean;
};

export type NavGroupConfig = {
  heading: string;
  items: NavItemConfig[];
};

export const workflow: NavGroupConfig = {
  heading: 'Workflow',
  items: [
    { label: 'Overview', route: '', Icon: OverviewIcon, exact: true },
    { label: 'Apps', route: 'apps', Icon: AppsIcon },
    { label: 'Functions', route: 'functions', Icon: FunctionsIcon },
    { label: 'Runs', route: 'runs', Icon: RunsIcon },
    { label: 'Event Types', route: 'event-types', Icon: EventsIcon },
    { label: 'Events', route: 'events', Icon: EventLogsIcon },
  ],
};

export const monitor: NavGroupConfig = {
  heading: 'Monitor',
  items: [
    { label: 'Metrics', route: 'metrics', Icon: MetricsIcon },
    { label: 'Insights', route: 'insights', Icon: InsightsIcon, beta: true },
  ],
};

export const experimentsItem: NavItemConfig = {
  label: 'Experiments',
  route: 'experiments',
  Icon: ExperimentsIcon,
  beta: true,
};

export const scoresItem: NavItemConfig = {
  label: 'Scores',
  route: 'scores',
  Icon: InsightsIcon,
  beta: true,
};

export const sessionsItem: NavItemConfig = {
  label: 'Sessions',
  route: 'sessions',
  Icon: SessionsIcon,
  beta: true,
};

export const manage: NavGroupConfig = {
  heading: 'Manage',
  items: [{ label: 'Webhooks', route: 'manage/webhooks', Icon: WebhooksIcon }],
};
