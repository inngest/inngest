import { useState } from 'react';

import CommandBlock, { type TabsProps } from '../CodeBlock/CommandBlock';
import { Link } from '../Link';
import { IconSpinner } from '../icons/Spinner';
import {
  DOCS_URL,
  INTRO_DESCRIPTION,
  STEPS,
  USE_CASES,
  VARIANT_TABS,
} from './experimentsEmptyStateContent';

// Fixed snippet viewport so the box keeps a stable height across tabs and the
// whole page stays compact; taller variants scroll internally.
const CODE_HEIGHT = 280;

function CodeExample({ tabs }: { tabs: TabsProps[] }) {
  const [activeTab, setActiveTab] = useState(tabs[0]?.title ?? '');
  const current = tabs.find((tab) => tab.title === activeTab) ?? tabs[0];

  return (
    <CommandBlock.Wrapper>
      <CommandBlock.Header className="flex items-center justify-between pr-2">
        <CommandBlock.Tabs tabs={tabs} activeTab={activeTab} setActiveTab={setActiveTab} />
        <CommandBlock.CopyButton content={current?.content} />
      </CommandBlock.Header>
      <CommandBlock currentTabContent={current} height={CODE_HEIGHT} />
    </CommandBlock.Wrapper>
  );
}

function Step({
  number,
  title,
  description,
  children,
}: {
  number: number;
  title: string;
  description: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <p className="text-basis text-sm font-medium">
          {number}. {title}
        </p>
        <p className="text-subtle text-sm leading-relaxed">{description}</p>
      </div>
      {children}
    </div>
  );
}

export function ExperimentsEmptyState() {
  return (
    <div className="bg-canvasBase flex flex-1 flex-col items-center overflow-auto px-6 py-12">
      <div className="mx-auto flex w-full max-w-[800px] flex-col gap-10">
        {/* Intro */}
        <div className="flex flex-col gap-2">
          <h1 className="text-basis text-2xl">Experiments</h1>
          <p className="text-subtle text-sm leading-relaxed">{INTRO_DESCRIPTION}</p>
          <Link href={DOCS_URL}>Learn more about experiments</Link>
        </div>

        {/* Empty-results indicator */}
        <div className="border-subtle text-subtle flex items-center gap-2 rounded-md border border-dashed px-4 py-3 text-sm">
          <IconSpinner className="fill-subtle h-4 w-4" />
          No experiments found
        </div>

        {/* Use cases */}
        <div className="grid grid-cols-2 gap-x-8 gap-y-6">
          {USE_CASES.map(({ Icon, title, description }) => (
            <div key={title} className="flex items-start gap-3">
              <div className="border-subtle bg-canvasSubtle text-basis flex h-10 w-10 shrink-0 items-center justify-center rounded-md border">
                <Icon className="h-5 w-5" />
              </div>
              <div className="flex flex-col gap-0.5">
                <p className="text-basis text-sm font-medium">{title}</p>
                <p className="text-muted text-sm leading-relaxed">{description}</p>
              </div>
            </div>
          ))}
        </div>

        <hr className="border-subtle" />

        {/* Get started */}
        <div className="flex flex-col gap-6">
          <h2 className="text-basis text-lg">Get started</h2>

          <Step number={1} title={STEPS.one.title} description={STEPS.one.description}>
            <CodeExample tabs={VARIANT_TABS} />
          </Step>

          <Step number={2} title={STEPS.two.title} description={STEPS.two.description} />
        </div>
      </div>
    </div>
  );
}
