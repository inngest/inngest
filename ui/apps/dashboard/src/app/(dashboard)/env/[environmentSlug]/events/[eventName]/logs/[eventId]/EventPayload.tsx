import { type JsonValue } from 'type-fest';

import SyntaxHighlighter from '@/components/SyntaxHighlighter';
import { getFragmentData, graphql, type FragmentType } from '@/gql';

const EventPayloadFragment = graphql(`
  fragment EventPayload on ArchivedEvent {
    payload: event
  }
`);

type EventPayloadProps = {
  event: FragmentType<typeof EventPayloadFragment>;
};

export default async function EventPayload({ event }: EventPayloadProps) {
  const { payload } = getFragmentData(EventPayloadFragment, event);

  let parsedPayload: string | JsonValue = '';
  if (typeof payload === 'string') {
    try {
      parsedPayload = JSON.parse(payload);
    } catch (error) {
      console.error(`Error parsing payload: `, error);
      parsedPayload = payload;
    }
  }

  const formattedPayload = JSON.stringify(parsedPayload, null, 2);

  return (
    <div className="flex h-full flex-col space-y-1.5 rounded-xl bg-slate-900 text-white">
      <header className="bg-slate-950 flex items-center justify-between rounded-t-xl p-1.5">
        <h3 className="px-6 py-2.5">Payload</h3>
      </header>
      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <SyntaxHighlighter language="json">{formattedPayload}</SyntaxHighlighter>
        </div>
      </div>
    </div>
  );
}
