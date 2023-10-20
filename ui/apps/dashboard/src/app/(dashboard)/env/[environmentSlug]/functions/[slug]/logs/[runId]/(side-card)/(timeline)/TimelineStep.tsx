'use client';

import { Disclosure } from '@headlessui/react';

import CheckmarkIcon from '@/icons/checkmark.svg';
import ChevronIcon from '@/icons/chevron.svg';
import RerunButton from './RerunButton';

type TimelineStepProps = {
  name: string;
  isCompleted?: boolean;
  children: React.ReactNode;
};

export default function TimelineStep({ name, isCompleted, children }: TimelineStepProps) {
  return (
    <div className="relative pt-8">
      <span className="absolute left-4 top-0 ml-px h-8 w-0.5 bg-slate-700" aria-hidden="true" />
      <Disclosure
        as="div"
        className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800 w-full rounded-lg bg-slate-800/50"
      >
        <div className="ui-open:rounded-b-none ui-not-open:text-slate-300 ui-open:border-slate-800 ui-open:border-b ui-open:text-sm flex w-full items-center rounded-lg bg-slate-800/50 text-left text-xs">
          <Disclosure.Button className="justify-left ui-open:p-4 flex flex-1 gap-3 p-1.5">
            <div className="ui-open:rotate-180 ui-open:transform ui-open:-ml-1.5 ml-1 flex h-4 w-4 items-center justify-center text-slate-500">
              <ChevronIcon />
            </div>
            <span className="capitalize">{name}</span>
          </Disclosure.Button>
          <div className="ui-open:p-4 flex items-center gap-4 p-1.5">
            {isCompleted && <CompletedStatus />}
            {/* <RerunButton /> */}
          </div>
        </div>
        <Disclosure.Panel>{children}</Disclosure.Panel>
      </Disclosure>
    </div>
  );
}

function CompletedStatus() {
  return (
    <div className="flex items-center gap-2">
      <div className="flex h-3 w-3 items-center justify-center text-teal-300">
        <CheckmarkIcon />
      </div>
      <span className="text-xs text-slate-400">Completed</span>
    </div>
  );
}
