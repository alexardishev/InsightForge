import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export interface Task {
  id: string;
  status: string;
  create_date: string;
  comment?: string | null;
}

interface TasksState {
  tasks: Task[];
  page: number;
  pageSize: number;
}

const initialState: TasksState = {
  tasks: [],
  page: 1,
  pageSize: 10,
};

const tasksSlice = createSlice({
  name: 'tasks',
  initialState,
  reducers: {
    setTasks(state, action: PayloadAction<Task[]>) {
      state.tasks = action.payload;
    },
    setPage(state, action: PayloadAction<number>) {
      state.page = action.payload;
    },
    setPageSize(state, action: PayloadAction<number>) {
      state.pageSize = action.payload;
    },
  },
});

export const { setTasks, setPage, setPageSize } = tasksSlice.actions;

export default tasksSlice.reducer;
