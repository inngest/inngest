import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

const devUrl = new URL(window.location.href);
devUrl.pathname = "";

export interface EventPayload {
  name: string;
}

export const devApi = createApi({
  reducerPath: "devApi",
  baseQuery: fetchBaseQuery({ baseUrl: devUrl.href }),
  endpoints: (builder) => ({
    sendEvent: builder.mutation<void, { id: string, name: string; ts: number, data?: object, user?: object }>({
      query: (event) => {
        let url = "/e/dev_key";
        /**
         * In dev mode, always assume that the dev server API is available at 8288. This
         * allows us to use a separate hot-reloading port for the UI when developing.
         *
         * NOTE: This has been removed to allow people to run `inngest dev --port 8111`:
         * users run multiple copies of inngest - one for dev, one for tests - as mock
         * environments.
         */
        if (import.meta.env.DEV) {
          const localDevUrl = new URL(url, devUrl);
          url = localDevUrl.href;
        }

        return {
          url,
          method: "POST",
          body: event,
        };
      },
    }),
  }),
});

export const { useSendEventMutation } = devApi;
export default devApi;
