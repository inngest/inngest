import { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';

import { getFunctionUsages, getFunctions } from '@/queries';
import type { FunctionState } from './FunctionStateFilter';
import {
  appendFunctionList,
  updateFunctionListWithUsage,
  type FunctionList,
} from './transformData';

type FunctionLists = { [key in FunctionState]: FunctionList };

const uninitializedFunctionList: FunctionList = {
  hasNextPage: false,
  isLoading: false,
  latestLoadedPage: 0,
  latestRequestedPage: 1,
  rows: [],
};

const initialFunctionLists: FunctionLists = {
  active: uninitializedFunctionList,
  archived: uninitializedFunctionList,
};

/**
 * Fetches function list data and returns the correct function list given the
 * specified function state.
 *
 * To request more data for a function list, increment its latestRequestedPage
 * field.
 */
export function useFunctionList({
  environmentID,
  onError,
  functionState,
}: {
  environmentID: string | undefined;
  functionState: FunctionState;
  onError: (error: Error) => void;
}): [FunctionList, (callback: (prev: FunctionList) => FunctionList) => void] {
  const client = useClient();
  const [functionLists, setFunctionLists] = useState(initialFunctionLists);
  const isArchived = functionState === 'archived';

  // A setter function that makes it easy to update a single function list.
  const setFunctionList: (callback: (prev: FunctionList) => FunctionList) => void = useCallback(
    (callback) => {
      setFunctionLists((prev) => {
        return {
          ...prev,
          [functionState]: callback(prev[functionState]),
        };
      });
    },
    [functionState]
  );

  useEffect(() => {
    // Missing environmentID means that the environment hasn't loaded yet.
    if (!environmentID) {
      return;
    }

    const functionList = functionLists[functionState];

    // This happens when the user toggles back-and-forth between "Active" and
    // "Archived".
    const isPageAlreadyLoaded = functionList.latestLoadedPage === functionList.latestRequestedPage;

    if (isPageAlreadyLoaded || functionList.isLoading) {
      return;
    }

    setFunctionList((prev) => {
      return {
        ...prev,
        isLoading: true,
      };
    });

    getFunctions({
      client,
      environmentID,
      isArchived,
      page: functionList.latestRequestedPage,
    }).then((res) => {
      if (res.error) {
        onError(res.error);
        return;
      }

      const { workspace } = res.data ?? {};
      if (!workspace) {
        onError(new Error('unable to load environment'));
        return;
      }

      setFunctionList((prev) => {
        let hasNextPage = false;
        if (typeof workspace.workflows.page.totalPages === 'number') {
          hasNextPage = workspace.workflows.page.totalPages > workspace.workflows.page.page;
        }

        return {
          ...appendFunctionList(prev, workspace.workflows),
          hasNextPage,
          isLoading: false,
          latestLoadedPage: prev.latestRequestedPage,
        };
      });

      // Since getFunctions doesn't return function usage data, we need to
      // separately fetch it. We do this because function usage data is more
      // expensive to fetch.
      getFunctionUsages({
        client,
        environmentID,
        isArchived,
        page: functionList.latestRequestedPage,
      }).then((res) => {
        const { workspace } = res.data ?? {};

        if (res.error || !workspace) {
          // Swallow error since the function list should still be usable
          // without usage data.
          return;
        }

        setFunctionList((prev) => {
          return updateFunctionListWithUsage(prev, workspace.workflows.data);
        });
      });
    });
  }, [environmentID, isArchived, functionState, functionLists, onError, setFunctionList, client]);

  return [functionLists[functionState], setFunctionList];
}
