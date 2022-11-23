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
    sendEvent: builder.mutation<void, any>({
      query: (rawPayload) => {
        /**
         * Assume we're being given something reasonable and try to prepare it.
         */
        const payload =
          typeof rawPayload === "string" ? JSON.parse(rawPayload) : rawPayload;
        delete payload.id;

        return {
          url: "/e/dev_key",
          method: "POST",
          body: JSON.stringify(payload),
        };
      },
    }),
  }),
});

export const { useSendEventMutation } = devApi;
export default devApi;
