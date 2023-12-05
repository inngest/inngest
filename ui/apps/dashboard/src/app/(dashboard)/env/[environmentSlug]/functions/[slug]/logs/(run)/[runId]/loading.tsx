import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { classNames } from '@inngest/components/utils/classNames';

export default function FunctionRunLoading() {
  return (
    <div className={classNames('dark grid h-full text-white', 'grid-cols-2')}>
      <EventDetails loading event={{}} />
      <RunDetails loading />
    </div>
  );
}
