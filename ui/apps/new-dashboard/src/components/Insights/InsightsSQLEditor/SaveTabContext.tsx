import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import { toast } from "sonner";

import { getIsSavedQuery } from "../InsightsTabManager/InsightsTabManager";
import { HOME_TAB, TEMPLATES_TAB } from "../InsightsTabManager/constants";
import { useStoredQueries } from "../QueryHelperPanel/StoredQueriesContext";
import type { Tab } from "../types";

type SaveTabContextValue = {
  saveTab: (tab: Tab) => Promise<void>;
  savingTabIds: ReadonlySet<string>;
};

const SaveTabContext = createContext<SaveTabContextValue | undefined>(
  undefined,
);

export function SaveTabProvider({ children }: { children: React.ReactNode }) {
  const { saveQuery } = useStoredQueries();
  const [savingTabIds, setSavingTabIds] = useState<Set<string>>(new Set());

  const saveTab = useCallback(
    async (tab: Tab) => {
      if (savingTabIds.has(tab.id)) return;

      const error = validateTab(tab);
      if (error) {
        toast.error(error);
        return;
      }

      setSavingTabIds((prev) => new Set(prev).add(tab.id));

      try {
        await saveQuery(tab);
      } finally {
        setSavingTabIds((prev) => {
          const next = new Set(prev);
          next.delete(tab.id);
          return next;
        });
      }
    },
    [saveQuery, setSavingTabIds, savingTabIds],
  );

  const value = useMemo<SaveTabContextValue>(
    () => ({ saveTab, savingTabIds }),
    [saveTab, savingTabIds],
  );

  return (
    <SaveTabContext.Provider value={value}>{children}</SaveTabContext.Provider>
  );
}

export function useSaveTabActions(): SaveTabContextValue {
  const ctx = useContext(SaveTabContext);
  if (!ctx)
    throw new Error("useSaveTabActions must be used within SaveTabProvider");

  return ctx;
}

export function useSaveTab(tab: Tab) {
  const { saveTab, savingTabIds } = useSaveTabActions();

  return useMemo(
    () => ({
      canSave: validateTab(tab) === null && !savingTabIds.has(tab.id),
      isSaved: getIsSavedQuery(tab),
      isSaving: savingTabIds.has(tab.id),
      saveTab: () => saveTab(tab),
    }),
    [saveTab, savingTabIds, tab],
  );
}

function validateTab(tab: Tab): null | string {
  if (!isQueryTab(tab)) return "Only query tabs can be saved.";
  if (tab.name === "") return "Unable to save query: name is required.";
  if (tab.query === "") return "Unable to save query: query is empty.";
  return null;
}

function isQueryTab(tab: Pick<Tab, "id">): boolean {
  return tab.id !== HOME_TAB.id && tab.id !== TEMPLATES_TAB.id;
}
