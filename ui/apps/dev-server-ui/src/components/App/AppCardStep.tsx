import { Disclosure, Transition } from '@headlessui/react';
import { classNames } from '@inngest/components/utils/classNames';

type AppCardStepProps = {
  isExpandable?: boolean;
  lineContent: React.ReactNode;
  expandedContent?: React.ReactNode;
  isEvenStep?: boolean;
};

export default function AppCardStep({
  isExpandable = true,
  lineContent,
  expandedContent,
  isEvenStep = false,
}: AppCardStepProps) {
  const verticalLineForOddStepsclassNames = `top-[2.52rem] left-[1.844rem] h-[calc(100%-2.52rem)]`;
  const verticalLineForEvenStepsclassNames = `top-0 left-[1.844rem] h-[1.22rem]`;
  return (
    <Disclosure
      as="div"
      className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800 relative"
    >
      <span
        className={classNames(
          `absolute w-px bg-slate-800`,
          isEvenStep ? verticalLineForEvenStepsclassNames : verticalLineForOddStepsclassNames
        )}
        aria-hidden="true"
      />
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
