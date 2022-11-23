import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";

const initialState: {
  contentView: "feed" | "docs";
  sidebarTab: "events" | "functions";
  selectedEvent: string | null;
  selectedRun: string | null;
} = {
  contentView: "feed",
  sidebarTab: "events",
  selectedEvent: null,
  selectedRun: null,
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
    selectContentView(
      state,
      action: PayloadAction<typeof initialState["contentView"]>
    ) {
      state.contentView = action.payload;
    },
  },
});

export const { selectEvent, selectRun, setSidebarTab, selectContentView } =
  globalState.actions;
export default globalState.reducer;
