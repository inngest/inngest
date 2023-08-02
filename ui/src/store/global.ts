import type { PayloadAction } from '@reduxjs/toolkit';
import { createSlice } from '@reduxjs/toolkit';

const initialState: {
  docsPath: string | null;
  selectedEvent: string | null;
  selectedRun: string | null;
} = {
  docsPath: null,
  selectedEvent: null,
  selectedRun: null,
};

const globalState = createSlice({
  name: 'global',
  initialState,
  reducers: {
    selectEvent(state, action: PayloadAction<string | null>) {
      state.selectedEvent = action.payload;
    },
    selectRun(state, action: PayloadAction<string | null>) {
      state.selectedRun = action.payload;
    },
    showDocs(state, action: PayloadAction<`/${string}` | null | undefined>) {
      if (typeof action.payload !== 'undefined') {
        state.docsPath = action.payload || null;
      }
    },
  },
});

export const { selectEvent, selectRun, showDocs } = globalState.actions;
export default globalState.reducer;
