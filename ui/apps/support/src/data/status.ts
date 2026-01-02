import { createServerFn } from "@tanstack/react-start";
import { getStatus as fetchInngestStatus } from "@/components/Support/Status";

export type { ExtendedStatus } from "@/components/Support/Status";

export const getStatus = createServerFn({ method: "GET" }).handler(async () => {
  return await fetchInngestStatus();
});
