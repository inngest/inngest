import { EventsIcon } from '@inngest/components/icons/sections/Events';

import { InlineCode } from '../Code';
import { TableBlankState } from '../Table/TableBlankState';

type TableBlankStateProps = {
  actions: React.ReactNode;
  title?: string;
};

export default function BlankState({ actions, title }: TableBlankStateProps) {
  return (
    <TableBlankState
      icon={<EventsIcon />}
      actions={actions}
      title={title || 'No events found'}
      description={
        <>
          To send events from within functions, you will use{' '}
          <InlineCode>step.sendEvent()</InlineCode>. This method takes a single event, or an array
          of events.
        </>
      }
    />
  );
}
