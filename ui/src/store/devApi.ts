import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { api, type Event } from './generated';

export interface EventPayload {
  name: string;
}

export const devApi = createApi({
  reducerPath: "devApi",
  baseQuery: fetchBaseQuery({ baseUrl: '/' }),
  endpoints: (builder) => ({
    sendEvent: builder.mutation<void, { id: string, name: string; ts: number, data?: object, user?: object }>({
      query: (event) => ({
        url: "/e/dev_key",
        method: "POST",
        body: event,
      }),
      onQueryStarted(event, { dispatch, queryFulfilled }) {
        // Optimistically add the event to the `GetEventQuery` cache so that it shows up in the UI
        // immediately.
        dispatch(
          api.util.upsertQueryData('GetEvent', { id: event.id }, {
            __typename: 'Query',
            event: {
              __typename: 'Event',
              id: event.id,
              name: event.name,
              raw: JSON.stringify(event),
              createdAt: event.ts,
            },
          }),
        )

        // Optimistically update the `GetEventsStreamQuery` cache with the new event so that it
        // shows up in the UI immediately.
        const patchEventsStreamsResult = dispatch(
          api.util.updateQueryData('GetEventsStream', undefined, (draftEvents) => {
            const normalizedEvent: Event = {
              __typename: 'Event',
              id: event.id,
              name: event.name,
              createdAt: event.ts,
            } as const
            if (draftEvents.events) {
              draftEvents.events.unshift(normalizedEvent);
            } else {
              draftEvents.events = [normalizedEvent];
            }
          }),
        )

        // If the event fails to send, undo the optimistic update.
        queryFulfilled.catch(patchEventsStreamsResult.undo);
      },
    }),
  }),
});

export const { useSendEventMutation } = devApi;
export default devApi;
