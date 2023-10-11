import { CheckIcon, ChevronDoubleRightIcon, XMarkIcon } from '@heroicons/react/20/solid';
import { capitalCase } from 'change-case';

import { Pill } from '@/components/Pill/Pill';

export default function DeployStatus({ status }: { status: string }): JSX.Element {
  // default is pending:
  let icon = <ChevronDoubleRightIcon className="-ml-1 h-3.5 w-3.5 text-yellow-500" />;
  switch (status) {
    case 'success':
      icon = <CheckIcon className="-ml-1 h-3.5 w-3.5 text-teal-500" />;
      break;
    case 'failed':
      icon = <XMarkIcon className="-ml-1 h-3.5 w-3.5 text-red-300" />;
  }

  return (
    <Pill variant="dark" className="gap-1">
      {icon} {capitalCase(status)}
    </Pill>
  );
}
