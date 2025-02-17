'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { Pill } from '@inngest/components/Pill/Pill';
import { Command } from 'cmdk';

type Props = {
  kind: 'app' | 'event' | 'eventType' | 'function' | 'run';
  onClick: () => unknown;
  path: Route;
  text: string;
  value: string;
};

export function ResultItem({ kind, onClick, path, text, value }: Props) {
  const router = useRouter();

  return (
    <Command.Item
      className="data-[selected]:bg-canvasSubtle/50 group flex cursor-pointer items-center rounded-md px-3 py-3"
      onSelect={() => {
        router.push(path);
        onClick();
      }}
      value={value}
    >
      <p className="flex-1 truncate">{text}</p>
      <Pill>{toKindName(kind)}</Pill>
    </Command.Item>
  );
}

function toKindName(kind: Props['kind']) {
  switch (kind) {
    case 'app':
      return 'App';
    case 'event':
      return 'Event';
    case 'eventType':
      return 'Event Type';
    case 'function':
      return 'Function';
    case 'run':
      return 'Run';
    default:
      return 'Unknown';
  }
}
