import { RiAlertLine, RiRouteLine, RiStackLine } from '@remixicon/react';

import { InlineCode } from '../Code';
import { FeatureEmptyState } from '../FeatureEmptyState';

type SessionsEmptyStateProps = {
  docsUrl?: string;
  onViewed?: () => void;
  onDocsLinkClick?: () => void;
  onPromptCopy?: () => void;
  onExampleCopy?: () => void;
};

const DEFAULT_DOCS_URL =
  'https://www.inngest.com/docs/features/events-triggers/sessions?ref=app-empty-sessions';

const EXAMPLE = `// Every run from this event joins the session
await inngest.send({
  name: 'app/message.created',
  data: { conversationId: 'conversation_1234' },
  meta: {
    sessions: { conversation_id: 'conversation_1234' },
  },
});`;

const PROMPT = `Read the docs about Inngest's sessions @https://www.inngest.com/docs/features/events-triggers/sessions and tell me how I can leverage them in my functions to group runs across conversations, threads, or multi-stage pipelines`;

const VALUE_PROPS = [
  {
    icon: RiRouteLine,
    iconClassName: '-scale-x-100',
    title: 'Trace a full flow',
    description: 'Every run tied to one conversation, ticket, or job.',
  },
  {
    icon: RiAlertLine,
    title: 'Catch failures fast',
    description: 'Run counts, failure rate, and last active per session.',
  },
  {
    icon: RiStackLine,
    title: 'Built for scale',
    description: 'Made for high-cardinality IDs you search repeatedly.',
  },
];

export function SessionsEmptyState({
  docsUrl = DEFAULT_DOCS_URL,
  onViewed,
  onDocsLinkClick,
  onPromptCopy,
  onExampleCopy,
}: SessionsEmptyStateProps) {
  return (
    <FeatureEmptyState
      title="Sessions"
      description="Group related runs into sessions. A session ties together every function run from the same conversation, job, or user flow, so you can find and inspect them all by one ID."
      docsUrl={docsUrl}
      onDocsLinkClick={onDocsLinkClick}
      valueProps={VALUE_PROPS}
      prompt={{
        description: 'Copy this prompt to learn about this feature and implement sessions',
        content: PROMPT,
        onCopy: onPromptCopy,
      }}
      example={{
        description: (
          <>
            add <InlineCode>meta.sessions</InlineCode> to any event
          </>
        ),
        tabs: [{ title: 'Code', content: EXAMPLE, readOnly: true, language: 'typescript' }],
        onCopy: onExampleCopy,
      }}
      onViewed={onViewed}
    />
  );
}
