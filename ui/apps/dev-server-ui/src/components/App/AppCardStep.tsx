import { Disclosure, Transition } from '@headlessui/react';

type AppCardStepProps = {
  isExpandable?: boolean;
  lineContent: React.ReactNode;
  expandedContent?: React.ReactNode;
};

export default function AppCardStep({
  isExpandable = true,
  lineContent,
  expandedContent,
}: AppCardStepProps) {
  return (
    <Disclosure
      as="div"
      className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800 relative"
    >
      <span className="absolute w-px bg-slate-800" aria-hidden="true" />
      <Disclosure.Button
        as={'div'}
        className="flex w-full cursor-pointer items-center justify-between p-4 pr-6 text-white"
      >
        {lineContent}
      </Disclosure.Button>
      {expandedContent && isExpandable && (
        <Transition
          enter="transition-opacity duration-200"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="transition-opacity duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Disclosure.Panel className="pb-4 pl-14 pr-6 text-gray-500 ">
            {expandedContent}
          </Disclosure.Panel>
        </Transition>
      )}
    </Disclosure>
  );
}
