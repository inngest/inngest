import { RunsIcon } from '@inngest/components/icons/sections/Runs';
import {
  RiAiGenerate,
  RiExchange2Line,
  RiListCheck3,
  RiNodeTree,
  RiShieldStarLine,
  RiSpyLine,
} from '@remixicon/react';
import type { ComponentType } from 'react';

type Item = {
  Icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
};

// Lead sentence is emphasized in the intro; the rest is regular weight.
export const INTRO_LEAD =
  'Run isolated code as a native step in your workflow.';
export const INTRO_REST =
  ' Sandbox execution is built directly into Inngest, so your code runs with the same durable execution, retries, tracing, and observability as every other step.';

// Bordered feature cards shown in a row under the intro.
export const FEATURES: Item[] = [
  {
    Icon: RiNodeTree,
    title: 'Native orchestration',
    description:
      'Run sandboxes within your workflow. No custom orchestration or glue code.',
  },
  {
    Icon: RiShieldStarLine,
    title: 'Durable by default',
    description:
      'Retries, state, and concurrency work out of the box. No additional configurations.',
  },
  {
    Icon: RunsIcon,
    title: 'Debug in one place',
    description:
      'Debug sandbox failures alongside the rest of your workflow from a single trace.',
  },
];

// "What can you build?" use cases, rendered as a 2-column grid.
export const USE_CASES: Item[] = [
  {
    Icon: RiAiGenerate,
    title: 'Run AI code safely',
    description: 'Execute AI-generated code in isolated environments',
  },
  {
    Icon: RiSpyLine,
    title: 'Isolate every agent',
    description: 'Give each user or agent its own sandbox.',
  },
  {
    Icon: RiListCheck3,
    title: 'Test with confidence',
    description: 'Run unit and integration tests in clean environments',
  },
  {
    Icon: RiExchange2Line,
    title: 'Clone and compare',
    description: 'Clone environments to test agent branches safely',
  },
];
