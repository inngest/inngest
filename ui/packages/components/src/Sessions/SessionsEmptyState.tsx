import { AccordionList, AccordionPrimitive } from '@inngest/components/AccordionCard/AccordionList';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { Link } from '@inngest/components/Link';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiAlertLine, RiArrowRightSLine, RiRouteLine, RiStackLine } from '@remixicon/react';

type SessionsEmptyStateProps = {
  docsUrl?: string;
};

const DEFAULT_DOCS_URL =
  'https://www.inngest.com/docs/features/events-triggers/sessions?ref=app-empty-sessions';

const example = `// Every run from this event joins the session
await inngest.send({
  name: 'app/message.created',
  data: { conversationId: 'conversation_1234' },
  meta: {
    sessions: { conversation_id: 'conversation_1234' },
  },
});`;

const prompt = `Read the docs about Inngest's sessions @https://www.inngest.com/docs/features/events-triggers/sessions and tell me how I can leverage them in my functions to group runs across conversations, threads, or multi-stage pipelines`;

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

export function SessionsEmptyState({ docsUrl = DEFAULT_DOCS_URL }: SessionsEmptyStateProps) {
  return (
    <div className="bg-canvasBase flex flex-1 flex-col items-center overflow-auto px-6 py-12">
      <div className="mx-auto flex w-full max-w-[800px] flex-col gap-10">
        <div className="flex flex-col gap-2">
          <h1 className="text-basis text-2xl">Sessions</h1>
          <p className="text-subtle text-sm leading-relaxed">
            Group related runs into sessions. A session ties together every function run from the
            same conversation, job, or user flow, so you can find and inspect them all by one ID.
          </p>
          <Link href={docsUrl} target="_blank">
            Learn more about sessions
          </Link>
        </div>

        <div className="border-subtle text-subtle flex items-center gap-2 rounded-md border border-dashed px-4 py-3 text-sm">
          <IconSpinner className="fill-subtle h-4 w-4" />
          Searching for sessions
        </div>

        <div className="grid grid-cols-2 gap-x-8 gap-y-6">
          {benefits.map(({ icon: Icon, iconClassName, title, description }) => (
            <div key={title} className="flex items-start gap-3">
              <div className="border-subtle bg-canvasSubtle text-basis flex h-10 w-10 shrink-0 items-center justify-center rounded-md border">
                <Icon className={`h-5 w-5 ${iconClassName ?? ''}`} />
              </div>
              <div className="flex flex-col gap-0.5">
                <p className="text-basis text-sm font-medium">{title}</p>
                <p className="text-muted text-sm leading-relaxed">{description}</p>
              </div>
            </div>
          ))}
        </div>

        <hr className="border-subtle" />

        <div className="flex flex-col gap-6">
          <h2 className="text-basis text-lg">Get started</h2>

          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between px-4 py-2.5">
              <p className="text-subtle text-sm">
                add <InlineCode>meta.sessions</InlineCode> to any event
              </p>
              <CommandBlock.CopyButton content={example} />
            </CommandBlock.Header>
            <CommandBlock
              currentTabContent={{
                title: 'Code',
                content: example,
                readOnly: true,
                language: 'typescript',
              }}
            />
          </CommandBlock.Wrapper>

          <AccordionList type="multiple" defaultValue={[]}>
            <AccordionList.Item value="prompt">
              <AccordionPrimitive.Header className="data-[state=open]:border-subtle flex items-center justify-between pr-4 text-sm data-[state=open]:border-b">
                <AccordionPrimitive.Trigger className="hover:bg-canvasSubtle group flex flex-1 items-center gap-1 px-3 py-4">
                  <RiArrowRightSLine className="h-5 w-5 transition-transform duration-200 group-data-[state=open]:rotate-90" />
                  <span className="text-subtle">Prompt for coding agent</span>
                </AccordionPrimitive.Trigger>
                <CommandBlock.CopyButton content={prompt} />
              </AccordionPrimitive.Header>
              <AccordionList.Content>
                <p className="text-muted whitespace-pre-wrap text-sm leading-5">{prompt}</p>
              </AccordionList.Content>
            </AccordionList.Item>
          </AccordionList>
        </div>
      </div>
    </div>
  );
}
