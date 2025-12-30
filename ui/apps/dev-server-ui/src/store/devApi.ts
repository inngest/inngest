import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { z } from 'zod';

import { api } from './generated';

const baseURL = import.meta.env.VITE_PUBLIC_API_BASE_URL
  ? new URL('/', import.meta.env.VITE_PUBLIC_API_BASE_URL)
  : '/';

export interface EventPayload {
  id?: string;
  name: string;
  ts?: number;
  data?: object;
  user?: object;
  functionId?: string;
}

const serverInfoSchema = z.object({
  version: z.string().optional(),
  isSingleNodeService: z.boolean().optional(),
  startOpts: z.record(z.unknown()).optional(),
});

export interface ServerInfo extends z.output<typeof serverInfoSchema> {
  isDiscoveryEnabled?: boolean;
}

export const devApi = createApi({
  reducerPath: 'devApi',
  baseQuery: fetchBaseQuery({ baseUrl: baseURL.toString() }),
  endpoints: (builder) => ({
    info: builder.query<ServerInfo, void>({
      query() {
        return {
          url: '/dev',
          method: 'GET',
        };
      },
      transformResponse(baseQueryReturnValue) {
        const info: ServerInfo = serverInfoSchema.parse(baseQueryReturnValue);

        if (info.startOpts) {
          info.isDiscoveryEnabled = Boolean(info.startOpts.autodiscover);
        }

        return info;
      },
    }),
    sendEvent: builder.mutation<void, EventPayload | EventPayload[]>({
      query: (payload) => {
        const isArray = Array.isArray(payload);
        const firstItem = isArray ? payload[0] : payload;
        const { functionId, ...body } = firstItem || {};

        return {
          url: functionId
            ? `/invoke/${encodeURIComponent(functionId)}`
            : '/e/dev_key',
          method: 'POST',
          body: isArray ? payload : body,
        };
      },
      onQueryStarted(event, { dispatch, queryFulfilled }) {
        // Don't optimistically update for arrays of events
        if (Array.isArray(event)) {
          return;
        }

        // Don't optimistically update if this is a function invocation, as the
        // shape of the payload will be different when sending vs receiving.
        if (event.functionId) {
          return;
        }

        // Don't optimistically update if the event doesn't have an ID
        if (!event.id) {
          return;
        }

        // Optimistically add the event to the `GetEventQuery` cache so that it shows up in the UI
        // immediately.
        dispatch(
          api.util.upsertQueryData(
            'GetEvent',
            { id: event.id },
            {
              __typename: 'Query',
              event: {
                __typename: 'Event',
                functionRuns: null,
                id: event.id,
                name: event.name,
                pendingRuns: null,
                raw: JSON.stringify(event),
                createdAt: event.ts as any,
                status: null,
              },
            },
          ),
        );
      },
    }),
  }),
});

export const { useSendEventMutation, useInfoQuery } = devApi;
export default devApi;
