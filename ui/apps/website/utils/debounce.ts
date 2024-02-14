import { useCallback } from "react";
import debounce from "lodash.debounce";

export const useDebounce = (func: (...args: any) => any, ms?: number) => {
  return useCallback(debounce(func, ms || 500), []);
};
