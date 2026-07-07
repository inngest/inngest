import { Inngest } from "inngest-v4";
import { endpointAdapter } from "inngest-v4/next";

export const inngestV4 = new Inngest({
  id: "test-suite-v4",
  endpointAdapter,
  isDev: true,
  baseUrl: process.env.INNGEST_BASE_URL ?? "http://127.0.0.1:8288",
});
