import type { PayloadAction } from '@reduxjs/toolkit';
import { createSlice } from '@reduxjs/toolkit';

const initialState: {
  selectedEvent: string | null;
  selectedRun: string | null;
} = {
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
  },
});

export const { selectEvent, selectRun } = globalState.actions;
export default globalState.reducer;
