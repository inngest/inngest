import { createServerFn } from "@tanstack/react-start";
import { getStatus as fetchInngestStatus } from "@inngest/components/Support/Status";

export type { ExtendedStatus } from "@inngest/components/Support/Status";

export const getStatus = createServerFn({ method: "GET" }).handler(async () => {
  return await fetchInngestStatus();
});
