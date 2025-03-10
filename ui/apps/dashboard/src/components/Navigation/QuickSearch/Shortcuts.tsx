import { cn } from '@inngest/components/utils/classNames';
import {
  RiAddLine,
  RiChat2Line,
  RiDiscordLine,
  RiExternalLinkLine,
  RiKey2Fill,
  RiMoonClearFill,
  RiPlugLine,
  RiSunLine,
  RiWindow2Line,
} from '@remixicon/react';
import { Command } from 'cmdk';
import { useTheme } from 'next-themes';

import { DISCORD_URL, pathCreator } from '@/utils/urls';
import { ResultItem } from './ResultItem';

export default function Shortcuts({
  onClose,
  envSlug,
  hasSearchTerm,
}: {
  onClose: () => void;
  envSlug: string;
  hasSearchTerm: boolean;
}) {
  const { theme, setTheme } = useTheme();
  return (
    <>
      <Command.Group
        heading={!hasSearchTerm ? 'Navigation' : undefined}
        className={cn(
          'text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1',
          hasSearchTerm && 'mb-0'
        )}
      >
        <ResultItem
          onClick={onClose}
          path={pathCreator.signingKeys({ envSlug })}
          text="Go to signing keys"
          value="nav-go-to-signing-keys"
          icon={<RiKey2Fill />}
        />
        <ResultItem
          onClick={onClose}
          path={pathCreator.keys({ envSlug })}
          text="Go to event keys"
          value="nav-go-to-event-keys"
          icon={<RiKey2Fill />}
        />
        <ResultItem
          onClick={onClose}
          path={pathCreator.integrations()}
          text="Go to Vercel"
          value="nav-go-to-vercel-integration"
          icon={<RiPlugLine />}
        />
        <ResultItem
          onClick={onClose}
          path={pathCreator.integrations()}
          text="Go to Neon"
          value="nav-go-to-neon-integration"
          icon={<RiPlugLine />}
        />
      </Command.Group>
      <Command.Group
        heading={!hasSearchTerm ? 'Actions' : undefined}
        className={cn(
          'text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1',
          hasSearchTerm && 'mb-0'
        )}
      >
        <ResultItem
          onClick={onClose}
          path={pathCreator.createApp({ envSlug })}
          text="Sync new app"
          value="action-sync-new-app"
          icon={<RiAddLine />}
        />
        {theme !== 'dark' && (
          <ResultItem
            onClick={() => {
              setTheme('dark');
              onClose();
            }}
            text="Switch to dark mode"
            value="action-dark-mode"
            icon={<RiMoonClearFill />}
          />
        )}
        {theme !== 'light' && (
          <ResultItem
            onClick={() => {
              setTheme('light');
              onClose();
            }}
            text="Switch to light mode"
            value="action-light-mode"
            icon={<RiSunLine />}
          />
        )}
        {theme !== 'system' && (
          <ResultItem
            onClick={() => {
              setTheme('system');
              onClose();
            }}
            text="Switch to system mode"
            value="action-system-mode"
            icon={<RiWindow2Line />}
          />
        )}
      </Command.Group>
      <Command.Group
        heading={!hasSearchTerm ? 'Help' : undefined}
        className={cn('text-muted text-xs [&_[cmdk-group-heading]]:mb-1', hasSearchTerm && 'mb-0')}
      >
        <ResultItem
          onClick={onClose}
          path={'https://www.inngest.com/docs?ref=app-cmdk'}
          text="Go to documentation"
          value="help-go-to-documentation-docs"
          icon={<RiExternalLinkLine />}
        />
        <ResultItem
          onClick={onClose}
          path={pathCreator.support()}
          text="Contact support"
          value="help-contact-support"
          icon={<RiChat2Line />}
        />
        <ResultItem
          onClick={onClose}
          path={DISCORD_URL}
          text="Join community"
          value="help-join-community"
          icon={<RiDiscordLine />}
        />
      </Command.Group>
    </>
  );
}
