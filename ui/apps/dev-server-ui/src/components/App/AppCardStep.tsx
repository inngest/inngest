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
    <Disclosure as="div" className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-subtle relative">
      <span className="bg-canvasBase absolute w-px" aria-hidden="true" />
      <Disclosure.Button
        as={'div'}
        className="text-basis flex w-full cursor-pointer items-center justify-between px-6 py-4"
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
          <Disclosure.Panel className="text-subtle pb-4 pl-14 pr-6">
            {expandedContent}
          </Disclosure.Panel>
        </Transition>
      )}
    </Disclosure>
  );
}
