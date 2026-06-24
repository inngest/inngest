import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock, { type TabsProps } from '@inngest/components/CodeBlock/CommandBlock';
import { Link } from '@inngest/components/Link';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import { RiAlertLine, RiExternalLinkLine, RiRouteLine, RiStackLine } from '@remixicon/react';

type SessionsEmptyStateProps = {
  docsUrl?: string;
  isLoading?: boolean;
};

const DEFAULT_DOCS_URL =
  // TODO: Replace with the sessions docs URL and include a ref param for dashboard traffic.
  'https://website-git-jakob-sessions-docs-inngest.vercel.app/docs/features/events-triggers/sessions';

const example = `// Every run from this event joins the session
await inngest.send({
  name: 'app/message.created',
  data: { conversationId: 'conversation_1234' },
  meta: {
    sessions: { conversation_id: 'conversation_1234' },
  },
});`;

const prompt = `Read the docs about Inngest's sessions @https://website-git-jakob-sessions-docs-inngest.vercel.app/docs-markdown/features/events-triggers/sessions and tell me how I can leverage them in my functions to group runs across conversations, threads, or multi-stage pipelines`;

const tabs: TabsProps[] = [
  {
    title: 'Code',
    content: example,
    readOnly: true,
    language: 'typescript',
  },
  {
    title: 'Prompt',
    content: prompt,
    readOnly: true,
    language: 'markdown',
    wordWrap: 'on',
  },
];

const defaultTab = tabs[0]!;

const benefits = [
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
  isLoading = false,
}: SessionsEmptyStateProps) {
  const [activeTab, setActiveTab] = useState(defaultTab.title);
  const currentTabContent = tabs.find((tab) => tab.title === activeTab) ?? defaultTab;

  return (
    <div className="bg-canvasBase text-basis flex flex-1 overflow-y-auto">
      <section className="mx-auto mt-16 flex w-full max-w-[816px] flex-col px-6 pb-16">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div className="border-muted bg-canvasSubtle flex h-12 w-12 shrink-0 items-center justify-center rounded-lg border">
              <SessionsIcon className="text-subtle h-6 w-6" />
            </div>
            <div className="flex flex-1 items-center gap-2">
              <h1 className="text-subtle text-2xl font-medium">Sessions</h1>
              {isLoading && (
                <div className="text-link flex items-center gap-1.5 text-sm">
                  <IconSpinner className="fill-link h-4 w-4" />
                  Waiting for your first session
                </div>
              )}
            </div>
            <div className="sm:ml-auto">
              <Button
                kind="primary"
                appearance="outlined"
                label="Go to docs"
                icon={<RiExternalLinkLine />}
                iconSide="left"
                href={docsUrl}
                target="_blank"
              />
            </div>
          </div>
          <p className="text-subtle max-w-[760px] text-base leading-6">
            <span className="text-basis text-lg">Group related runs into sessions.</span>
            <br />A session ties together every function run from the same conversation, job, or
            user flow, so you can find and inspect them all by one ID.{' '}
            <Link href={docsUrl} target="_blank" size="small" className="inline-block">
              Learn more about sessions
            </Link>
          </p>
        </div>

        <div className="border-subtle bg-canvasBase mt-6 grid grid-cols-1 gap-6 rounded-md border p-4 md:grid-cols-3">
          {benefits.map(({ icon: Icon, iconClassName, title, description }) => (
            <div key={title} className="flex flex-col gap-2">
              <Icon className={`text-subtle h-6 w-6 ${iconClassName ?? ''}`} />
              <div className="flex flex-col gap-1">
                <h2 className="text-basis truncate text-base font-medium">{title}</h2>
                <p className="text-muted text-sm leading-5">{description}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col gap-2">
          <div>
            <h2 className="text-basis text-lg font-medium">Get started</h2>
            <p className="text-subtle text-base leading-6">
              Add <InlineCode>meta.sessions</InlineCode> to any event
            </p>
          </div>
          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between pr-4">
              <CommandBlock.Tabs tabs={tabs} activeTab={activeTab} setActiveTab={setActiveTab} />
              <CommandBlock.CopyButton content={currentTabContent.content} />
            </CommandBlock.Header>
            <CommandBlock currentTabContent={currentTabContent} />
          </CommandBlock.Wrapper>
        </div>
      </section>
    </div>
  );
}
