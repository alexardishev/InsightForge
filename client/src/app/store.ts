import { configureStore } from '@reduxjs/toolkit';
import settingsReducer from '../features/settings/settingsSlice';
import viewBuilderReducer from '../features/viewBuilder/viewBuilderSlice';

export const store = configureStore({
  reducer: {
    settings: settingsReducer,
    viewBuilder: viewBuilderReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
