import { useMemo } from "react";

/**
 * Given a JSON string, returns a pretty-printed version of it if it's valid
 * JSON, else returns `null`.
 */
export const usePrettyJson = (
  json: string | null | undefined
): string | null => {
  return useMemo(() => {
    try {
      const data = JSON.parse(json as string);
      if (data === null) {
        throw new Error();
      }

      return JSON.stringify(data, null, 2);
    } catch (e) {
      console.warn("Unable to parse content as JSON: ", json);
      return "";
    }
  }, [json]);
};
