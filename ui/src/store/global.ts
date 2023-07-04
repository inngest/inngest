import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";

const initialState: {
  contentView: "feed" | "functions" | "docs" | "apps";
  docsPath: string | null;
  sidebarTab: "events" | "functions";
  selectedEvent: string | null;
  selectedRun: string | null;
  showingSendEventModal: boolean;
  sendEventModalData: string | null;
} = {
  contentView: "feed",
  docsPath: null,
  sidebarTab: "events",
  selectedEvent: null,
  selectedRun: null,
  showingSendEventModal: false,
  sendEventModalData: null,
};

const globalState = createSlice({
  name: "global",
  initialState,
  reducers: {
    setSidebarTab(
      state,
      action: PayloadAction<typeof initialState["sidebarTab"]>
    ) {
      state.sidebarTab = action.payload;
    },
    selectEvent(state, action: PayloadAction<string | null>) {
      state.selectedEvent = action.payload;
    },
    selectRun(state, action: PayloadAction<string | null>) {
      state.selectedRun = action.payload;
    },
    showFeed(state) {
      state.contentView = "feed";
    },
    showApps(state) {
      state.contentView = "apps";
    },
    showFunctions(state) {
      state.contentView = "functions";
    },
    showDocs(state, action: PayloadAction<`/${string}` | null | undefined>) {
      state.contentView = "docs";

      if (typeof action.payload !== "undefined") {
        state.docsPath = action.payload || null;
      }
    },
    showEventSendModal: (
      state,
      action: PayloadAction<{ show: boolean; data?: string | null }>
    ) => {
      state.showingSendEventModal = action.payload.show;

      if (typeof action.payload.data !== "undefined") {
        state.sendEventModalData = action.payload.data;
      }
    },
  },
});

export const {
  selectEvent,
  selectRun,
  setSidebarTab,
  showDocs,
  showFunctions,
  showFeed,
  showApps,
  showEventSendModal,
} = globalState.actions;
export default globalState.reducer;
