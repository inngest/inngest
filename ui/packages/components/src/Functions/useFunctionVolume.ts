import { useQuery } from '@tanstack/react-query';

import type { FunctionsTable } from './FunctionsTable';

export function useFunctionVolume(
  functionID: string,
  getFunctionVolume: React.ComponentProps<typeof FunctionsTable>['getFunctionVolume']
) {
  return useQuery({
    queryKey: ['function-volume', functionID],
    queryFn: () => getFunctionVolume({ functionID }),
    staleTime: 5 * 60 * 1000, // cache for 5 min
  });
}
