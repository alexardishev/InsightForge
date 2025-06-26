import { configureStore } from '@reduxjs/toolkit';
import settingsReducer from '../features/settings/settingsSlice';
import viewBuilderReducer from '../features/viewBuilder/viewBuilderSlice';
import tasksReducer from '../features/tasks/tasksSlice';

export const store = configureStore({
  reducer: {
    settings: settingsReducer,
    viewBuilder: viewBuilderReducer,
    tasks: tasksReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
