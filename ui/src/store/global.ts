import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";

const initialState: {
  sidebarTab: "events" | "functions";
  selectedEvent: string | null;
  selectedHistoryEvent: string | null;
} = {
  sidebarTab: "events",
  selectedEvent: null,
  selectedHistoryEvent: null,
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
    selectHistoryEvent(state, action: PayloadAction<string | null>) {
      state.selectedHistoryEvent = action.payload;
    },
  },
});

export const { selectEvent, selectHistoryEvent, setSidebarTab } =
  globalState.actions;
export default globalState.reducer;
