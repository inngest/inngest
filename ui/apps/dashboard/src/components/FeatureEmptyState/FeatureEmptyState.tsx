import {
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
} from 'react';

import CommandBlock, {
  type TabsProps,
} from '@inngest/components/CodeBlock/CommandBlock';
import { Link } from '@inngest/components/Link';

import {
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
  type AnalyticsFeature,
} from '@/utils/analyticsEvents';

export type FeatureEmptyStateValueProp = {
  icon: ComponentType<{ className?: string }>;
  iconClassName?: string;
  title: string;
  description: string;
};

export type FeatureEmptyStateProps = {
  feature: AnalyticsFeature;
  title: string;
  description: ReactNode;
  docsUrl: string;
  onDocsLinkClick?: () => void;
  valueProps: FeatureEmptyStateValueProp[];
  prompt: {
    description: ReactNode;
    content: string;
  };
  example: {
    description?: ReactNode;
    tabs: TabsProps[];
    height?: number;
  };
  // Renders a small bordered card (title, description, and the copyable
  // prompt only — no value props grid, no code example) instead of the
  // full immersive page — for callers that show this alongside other
  // content (e.g. a banner above a dashboard's own skeleton/empty charts)
  // rather than in place of it.
  compact?: boolean;
  className?: string;
};

const PROMPT_HEIGHT = 124;
const COMPACT_PROMPT_HEIGHT = 80;

function ValuePropItem({
  icon: Icon,
  iconClassName,
  title,
  description,
}: FeatureEmptyStateValueProp) {
  return (
    <div className="flex items-start gap-3">
      <div className="border-subtle bg-canvasSubtle text-basis flex h-10 w-10 shrink-0 items-center justify-center rounded-md border">
        <Icon className={`h-5 w-5 ${iconClassName ?? ''}`} />
      </div>
      <div className="flex flex-col gap-0.5">
        <p className="text-basis text-sm font-medium">{title}</p>
        <p className="text-muted text-sm leading-relaxed">{description}</p>
      </div>
    </div>
  );
}

function ExampleBlock({
  description,
  tabs,
  height,
  feature,
}: FeatureEmptyStateProps['example'] & { feature: AnalyticsFeature }) {
  const [activeTab, setActiveTab] = useState(tabs[0]?.title ?? '');
  const current = tabs.find((tab) => tab.title === activeTab) ?? tabs[0];
  const hasTabs = tabs.length > 1;

  return (
    <CommandBlock.Wrapper>
      <CommandBlock.Header
        className={
          hasTabs
            ? 'flex items-center justify-between pr-2'
            : 'flex items-center justify-between px-4 py-2.5'
        }
      >
        {hasTabs ? (
          <CommandBlock.Tabs
            tabs={tabs}
            activeTab={activeTab}
            setActiveTab={setActiveTab}
          />
        ) : (
          <p className="text-subtle text-sm">{description}</p>
        )}
        <CommandBlock.CopyButton
          content={current?.content}
          onCopy={() => trackEmptyStateExampleCopied({ feature })}
        />
      </CommandBlock.Header>
      <CommandBlock currentTabContent={current} height={height} />
    </CommandBlock.Wrapper>
  );
}

export function FeatureEmptyState({
  feature,
  title,
  description,
  docsUrl,
  onDocsLinkClick,
  valueProps,
  prompt,
  example,
  compact = false,
  className,
}: FeatureEmptyStateProps) {
  // Fire once on view. The ref guards against React 18 StrictMode's
  // double-invoke so we don't double-count.
  const viewedRef = useRef(false);
  useEffect(() => {
    if (viewedRef.current) return;
    viewedRef.current = true;
    trackEmptyStateViewed({ feature });
  }, [feature]);

  if (compact) {
    return (
      <div className={`border-subtle bg-canvasBase flex flex-col gap-3 rounded-md border p-4 ${className ?? ''}`}>
        <div className="flex flex-col gap-3">
          <h2 className="text-basis text-base font-medium">{title}</h2>
          <p className="text-muted text-sm leading-relaxed">{description}</p>
        </div>

        <CommandBlock.Wrapper>
          <CommandBlock.Header className="flex items-center justify-between px-4 py-2.5">
            <p className="text-subtle text-sm">{prompt.description}</p>
            <CommandBlock.CopyButton
              content={prompt.content}
              onCopy={() => trackEmptyStatePromptCopied({ feature })}
            />
          </CommandBlock.Header>
          <CommandBlock
            height={COMPACT_PROMPT_HEIGHT}
            currentTabContent={{
              title: 'Prompt',
              content: prompt.content,
              readOnly: true,
              language: 'plaintext',
              wordWrap: 'on',
            }}
          />
        </CommandBlock.Wrapper>
      </div>
    );
  }

  return (
    <div className="bg-canvasBase flex flex-1 flex-col items-center overflow-auto px-6 py-12">
      <div className="mx-auto flex w-full max-w-[800px] flex-col gap-10">
        <div className="flex flex-col gap-2">
          <h1 className="text-basis text-2xl">{title}</h1>
          <p className="text-subtle text-sm leading-relaxed">{description}</p>
          <Link href={docsUrl} target="_blank" onClick={onDocsLinkClick}>
            Learn more about {title.toLowerCase()}
          </Link>
        </div>

        <div className="grid grid-cols-2 gap-x-8 gap-y-6">
          {valueProps.map((valueProp) => (
            <ValuePropItem key={valueProp.title} {...valueProp} />
          ))}
        </div>

        <hr className="border-subtle" />

        <div className="flex flex-col gap-6">
          <h2 className="text-basis text-lg">Get started</h2>

          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between px-4 py-2.5">
              <p className="text-subtle text-sm">{prompt.description}</p>
              <CommandBlock.CopyButton
                content={prompt.content}
                onCopy={() => trackEmptyStatePromptCopied({ feature })}
              />
            </CommandBlock.Header>
            <CommandBlock
              height={PROMPT_HEIGHT}
              currentTabContent={{
                title: 'Prompt',
                content: prompt.content,
                readOnly: true,
                language: 'plaintext',
                wordWrap: 'on',
              }}
            />
          </CommandBlock.Wrapper>

          <ExampleBlock {...example} feature={feature} />
        </div>
      </div>
    </div>
  );
}
