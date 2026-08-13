import { useEffect, useRef } from 'react';

import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import {
  FeatureEmptyState as SharedFeatureEmptyState,
  type FeatureEmptyStateProps as SharedFeatureEmptyStateProps,
  type FeatureEmptyStateValueProp,
} from '@inngest/components/FeatureEmptyState/FeatureEmptyState';

import {
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
  type AnalyticsFeature,
} from '@/utils/analyticsEvents';

export type { FeatureEmptyStateValueProp };

export type FeatureEmptyStateProps = Omit<
  SharedFeatureEmptyStateProps,
  'onView' | 'onPromptCopy' | 'onExampleCopy' | 'contentOverride'
> & {
  feature: AnalyticsFeature;
  // Renders a small bordered card (title, description, and the copyable
  // prompt only — no value props grid, no code example) instead of the
  // full immersive page — for callers that show this alongside other
  // content (e.g. a banner above a dashboard's own skeleton/empty charts)
  // rather than in place of it.
  compact?: boolean;
  className?: string;
};

const COMPACT_PROMPT_HEIGHT = 80;

function CompactEmptyState({
  feature,
  title,
  description,
  prompt,
  className,
}: Pick<
  FeatureEmptyStateProps,
  'feature' | 'title' | 'description' | 'prompt' | 'className'
>) {
  // Fire once on view. The ref guards against React 18 StrictMode's
  // double-invoke so we don't double-count.
  const viewedRef = useRef(false);
  useEffect(() => {
    if (viewedRef.current) return;
    viewedRef.current = true;
    trackEmptyStateViewed({ feature });
  }, [feature]);

  return (
    <div
      className={`border-subtle bg-canvasBase flex flex-col gap-3 rounded-md border p-4 ${
        className ?? ''
      }`}
    >
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

// Dashboard wrapper around the shared component: maps the feature slug to
// the Segment tracking calls so consumers keep the pre-extraction API and
// event surface. The compact variant is only used by the dashboard's AI
// Overview page, so it lives here rather than in the shared package.
export function FeatureEmptyState({
  feature,
  compact = false,
  className,
  ...props
}: FeatureEmptyStateProps) {
  if (compact) {
    return (
      <CompactEmptyState
        feature={feature}
        title={props.title}
        description={props.description}
        prompt={props.prompt}
        className={className}
      />
    );
  }

  return (
    <SharedFeatureEmptyState
      {...props}
      onView={() => trackEmptyStateViewed({ feature })}
      onPromptCopy={() => trackEmptyStatePromptCopied({ feature })}
      onExampleCopy={() => trackEmptyStateExampleCopied({ feature })}
    />
  );
}
