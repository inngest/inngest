import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

import.meta.env.DEV;

const devUrl = new URL(window.location.href);
devUrl.pathname = "";

export interface EventPayload {
  name: string;
}

export const devApi = createApi({
  reducerPath: "devApi",
  baseQuery: fetchBaseQuery({ baseUrl: devUrl.href }),
  endpoints: (builder) => ({
    sendEvent: builder.mutation<void, any>({
      query: (rawPayload) => {
        /**
         * Assume we're being given something reasonable and try to prepare it.
         */
        const payload =
          typeof rawPayload === "string" ? JSON.parse(rawPayload) : rawPayload;
        delete payload.id;

        let url = "/e/dev_key";

        /**
         * In dev mode, always assume that the dev server API is available at 8288. This
         * allows us to use a separate hot-reloading port for the UI when developing.
         */
        if (import.meta.env.DEV) {
          const localDevUrl = new URL(url, devUrl);
          localDevUrl.port = "8288";
          url = localDevUrl.href;
        }

        return {
          url,
          method: "POST",
          body: JSON.stringify(payload),
        };
      },
    }),
  }),
});

export const { useSendEventMutation } = devApi;
export default devApi;
