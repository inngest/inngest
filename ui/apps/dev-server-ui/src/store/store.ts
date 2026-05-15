import { configureStore } from '@reduxjs/toolkit';
import devApi from './devApi';
import { api } from './generated';

export const store = configureStore({
  reducer: {
    [api.reducerPath]: api.reducer,
    [devApi.reducerPath]: devApi.reducer,
  },
  middleware: (getDefaultMiddleware) => {
    const defaults = getDefaultMiddleware();

    //
    // Fix CommonJS/ESM interop issue where redux-thunk gets imported as { default: fn }
    const fixedDefaults = defaults.map((m: any) =>
      typeof m === 'object' && m.default && typeof m.default === 'function'
        ? m.default
        : m,
    );

    return fixedDefaults.concat(api.middleware, devApi.middleware);
  },
});

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>;
// Inferred type: {posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch;
