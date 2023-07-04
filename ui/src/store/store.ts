import { configureStore } from '@reduxjs/toolkit';
import devApi from './devApi';
import { api } from './generated';

export const store = configureStore({
  reducer: {
    [api.reducerPath]: api.reducer,
    [devApi.reducerPath]: devApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(api.middleware, devApi.middleware),
});

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>;
