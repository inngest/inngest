import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { toast } from "sonner";

import type { TabManagerActions } from "@/components/Insights/InsightsTabManager/InsightsTabManager";
import type { QuerySnapshot, Tab } from "@/components/Insights/types";
import type { InsightsQueryStatement } from "@/gql/graphql";
import { getOrderedSavedQueries } from "../queries";
import { useInsightsSavedQueries } from "./useInsightsSavedQueries";

interface StoredQueriesContextValue {
  deleteQuery: (queryId: string) => void;
  deleteQuerySnapshot: (snapshotId: string) => void;
  isSavedQueriesFetching: boolean;
  queries: {
    data: undefined | InsightsQueryStatement[];
    error: undefined | string;
    isLoading: boolean;
  };
  querySnapshots: {
    data: QuerySnapshot[];
    error: undefined;
    isLoading: boolean;
  };
  saveQuery: (tab: Tab) => Promise<void>;
  saveQuerySnapshot: (snapshot: QuerySnapshot) => void;
  shareQuery: (queryId: string) => void;
}

const StoredQueriesContext = createContext<
  undefined | StoredQueriesContextValue
>(undefined);

interface StoredQueriesProviderProps {
  children: ReactNode;
  tabManagerActions: TabManagerActions;
}

export function StoredQueriesProvider({
  children,
  tabManagerActions,
}: StoredQueriesProviderProps) {
  const [querySnapshots, setQuerySnapshots] = useState<QuerySnapshot[]>([]);

  const {
    deleteQuery: beDeleteQuery,
    savedQueries: beSavedQueries,
    savedQueriesError,
    isSavedQueriesFetching,
    saveQuery: beSaveQuery,
    shareQuery: beShareQuery,
    updateQuery: beUpdateQuery,
    refetchSavedQueries,
  } = useInsightsSavedQueries();

  const saveQuery = useCallback(
    async (tab: Tab) => {
      if (tab.savedQueryId !== undefined) {
        const result = await beUpdateQuery({
          id: tab.savedQueryId,
          name: tab.name,
          query: tab.query,
        });
        if (result.ok) {
          // TODO: This often leads to double-fetching, but it's currently needed because the "InsightsQueryStatement"
          // __typename does not exist and does not auto-refetch if the list was previously empty. We need to make sure
          // that we have a consistent type name to match on regardless of existing saved queries.
          refetchSavedQueries();
          toast.success("Successfully updated query");
        } else {
          const errorMessage = `Failed to update query${
            result.error === "unique" ? ": name must be unique" : ""
          }`;
          toast.error(errorMessage);
          throw new Error(errorMessage);
        }
      } else {
        const result = await beSaveQuery({ name: tab.name, query: tab.query });
        if (result.ok) {
          tabManagerActions.updateTab(tab.id, { savedQueryId: result.data.id });
          // TODO: This often leads to double-fetching, but it's currently needed because the "InsightsQueryStatement"
          // __typename does not exist and does not auto-refetch if the list was previously empty. We need to make sure
          // that we have a consistent type name to match on regardless of existing saved queries.
          refetchSavedQueries();
          toast.success("Successfully saved query");
        } else {
          const errorMessage = `Failed to save query${
            result.error === "unique" ? ": name must be unique" : ""
          }`;
          toast.error(errorMessage);
          throw new Error(errorMessage);
        }
      }
    },
    [beSaveQuery, beUpdateQuery, refetchSavedQueries, tabManagerActions],
  );

  const deleteQuery = useCallback(
    async (queryId: string) => {
      const result = await beDeleteQuery({ id: queryId });
      if (result.ok) {
        tabManagerActions.breakQueryAssociation(queryId);
        // This is necessary because the query never returns anything that matches the list by __typename.
        // It returns only a list of deleted IDs.
        refetchSavedQueries();
        toast.success("Query deleted");
      } else {
        toast.error("Failed to delete query");
      }
    },
    [beDeleteQuery, refetchSavedQueries, tabManagerActions],
  );

  const shareQuery = useCallback(
    async (queryId: string) => {
      const result = await beShareQuery({ id: queryId });
      if (result.ok) {
        // TODO: This often leads to double-fetching, but it's currently needed because the "InsightsQueryStatement"
        // __typename does not exist and does not auto-refetch if the list was previously empty. We need to make sure
        // that we have a consistent type name to match on regardless of existing saved queries.
        refetchSavedQueries();
        toast.success("Query shared with your organization");
      } else {
        toast.error("Failed to share query with your organization");
      }
    },
    [beShareQuery, refetchSavedQueries],
  );

  const deleteQuerySnapshot = useCallback((snapshotId: string) => {
    setQuerySnapshots((prev) => prev.filter((s) => s.id !== snapshotId));
  }, []);

  const saveQuerySnapshot = useCallback((snapshot: QuerySnapshot) => {
    setQuerySnapshots((current) => [snapshot, ...current].slice(0, 10));
  }, []);

  const queries = useMemo(() => {
    return {
      data: getOrderedSavedQueries(beSavedQueries),
      error: savedQueriesError ? savedQueriesError.message : undefined,
      isLoading: isSavedQueriesFetching,
    };
  }, [beSavedQueries, isSavedQueriesFetching, savedQueriesError]);

  const orderedQuerySnapshots = useMemo(
    () => ({ data: querySnapshots, error: undefined, isLoading: false }),
    [querySnapshots],
  );

  return (
    <StoredQueriesContext.Provider
      value={{
        deleteQuery,
        deleteQuerySnapshot,
        isSavedQueriesFetching,
        shareQuery,
        queries,
        querySnapshots: orderedQuerySnapshots,
        saveQuery,
        saveQuerySnapshot,
      }}
    >
      {children}
    </StoredQueriesContext.Provider>
  );
}

export function useStoredQueries(): StoredQueriesContextValue {
  const context = useContext(StoredQueriesContext);
  if (context === undefined) {
    throw new Error(
      "useStoredQueries must be used within a StoredQueriesProvider",
    );
  }

  return context;
}
