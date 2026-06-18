import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import { Search } from '@inngest/components/Forms/Search';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import { RiExternalLinkLine, RiSearchLine } from '@remixicon/react';

type SessionsEmptyStateProps = {
  onSubmit: (sessionKey: string) => void;
  docsUrl?: string;
};

export function SessionsEmptyState({
  onSubmit,
  // TODO: Replace with the sessions docs URL and include a ref param for dashboard traffic.
  docsUrl = 'https://website-git-jakob-sessions-docs-inngest.vercel.app/docs/features/events-triggers/sessions',
}: SessionsEmptyStateProps) {
  const [value, setValue] = useState('');
  const trimmed = value.trim();

  return (
    <div className="bg-canvasBase flex flex-1 flex-col items-center justify-center px-6">
      <form
        className="flex w-full max-w-[640px] flex-col items-center gap-6"
        onSubmit={(e) => {
          e.preventDefault();
          if (trimmed) {
            onSubmit(trimmed);
          }
        }}
      >
        <div className="border-subtle bg-canvasSubtle flex h-12 w-12 items-center justify-center rounded-lg border">
          <SessionsIcon className="text-basis h-6 w-6" />
        </div>
        <p className="text-subtle max-w-md text-center text-sm leading-relaxed">
          Sessions group runs that share a session key (eg. <InlineCode>conversation_id</InlineCode>
          ). Search by session key below:
        </p>
        <Search
          name="sessionKey"
          placeholder="Search by session key"
          value={value}
          maxLength={128}
          autoFocus
          className="w-full"
          onUpdate={setValue}
        />
        <div className="flex items-center justify-center gap-3">
          <Button
            appearance="outlined"
            label="Go to docs"
            icon={<RiExternalLinkLine />}
            iconSide="left"
            href={docsUrl}
            target="_blank"
          />
          <Button
            type="submit"
            kind="primary"
            label="Search sessions"
            icon={<RiSearchLine />}
            iconSide="left"
            disabled={!trimmed}
          />
        </div>
      </form>
    </div>
  );
}
