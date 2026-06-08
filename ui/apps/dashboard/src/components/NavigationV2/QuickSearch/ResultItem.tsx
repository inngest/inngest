import { Pill } from '@inngest/components/Pill';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RunsIcon } from '@inngest/components/icons/sections/Runs';
import { RiQuestionMark } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';
import { Command } from 'cmdk';

type Props = {
  isDifferentEnv?: boolean;
  kind?: 'app' | 'event' | 'eventType' | 'function' | 'run';
  onClick: () => unknown;
  path?: string;
  text: string;
  value: string;
  icon?: React.ReactNode;
};

function isExternalUrl(path: string): boolean {
  return /^https?:\/\//.test(path);
}

export function ResultItem({
  isDifferentEnv = false,
  kind,
  onClick,
  path,
  text,
  value,
  icon,
}: Props) {
  const navigate = useNavigate();

  return (
    <Command.Item
      className="data-[selected=true]:bg-canvasSubtle/50 text-basis group flex h-10 cursor-pointer items-center gap-2 rounded-md px-2 text-sm"
      onSelect={() => {
        if (path) {
          if (isExternalUrl(path)) {
            // Off-app links (docs, support, Discord) must go through the
            // browser; the router's navigate() only resolves in-app routes.
            window.open(path, '_blank', 'noopener,noreferrer');
          } else {
            navigate({ to: path });
          }
        }
        onClick();
      }}
      value={value}
      // Filter on the visible text too, so a server match shown by name but
      // keyed by id (e.g. events) isn't hidden by cmdk's client-side filter.
      keywords={[text]}
    >
      <span className="text-light flex h-4 w-4 items-center justify-center">
        {kind ? getKindDetails(kind).icon : icon}
      </span>
      <p className="flex-1 truncate">{text}</p>

      {isDifferentEnv && (
        <span className="h-5">
          <Pill appearance="solidBright" className="mb-3" kind="warning">
            Different environment
          </Pill>
        </span>
      )}
    </Command.Item>
  );
}

function getKindDetails(kind: Props['kind']) {
  switch (kind) {
    case 'app':
      return { name: 'App', icon: <AppsIcon /> };
    case 'event':
      return { name: 'Event', icon: <EventsIcon /> };
    case 'eventType':
      return { name: 'Event Type', icon: <EventLogsIcon /> };
    case 'function':
      return { name: 'Function', icon: <FunctionsIcon /> };
    case 'run':
      return { name: 'Run', icon: <RunsIcon /> };
    default:
      return { name: 'Unknown', icon: <RiQuestionMark /> };
  }
}
