import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SelectedColumn {
  table: string;
  column: string;
}

interface ViewBuilderState {
  selectedDb: string;
  selectedSchema: string;
  selectedTables: string[];
  selectedColumns: SelectedColumn[];
}

const initialState: ViewBuilderState = {
  selectedDb: '',
  selectedSchema: '',
  selectedTables: [],
  selectedColumns: [],
};

const viewBuilderSlice = createSlice({
  name: 'viewBuilder',
  initialState,
  reducers: {
    setSelectedDb(state, action: PayloadAction<string>) {
      state.selectedDb = action.payload;
      state.selectedSchema = '';
      state.selectedTables = [];
      state.selectedColumns = [];
    },
    setSelectedSchema(state, action: PayloadAction<string>) {
      state.selectedSchema = action.payload;
      state.selectedTables = [];
      state.selectedColumns = [];
    },
    toggleTable(state, action: PayloadAction<string>) {
      const table = action.payload;
      if (state.selectedTables.includes(table)) {
        state.selectedTables = state.selectedTables.filter((t) => t !== table);
        state.selectedColumns = state.selectedColumns.filter((c) => c.table !== table);
      } else {
        state.selectedTables.push(table);
      }
    },
    toggleColumn(state, action: PayloadAction<SelectedColumn>) {
      const { table, column } = action.payload;
      const exists = state.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (exists) {
        state.selectedColumns = state.selectedColumns.filter(
          (c) => !(c.table === table && c.column === column),
        );
      } else {
        state.selectedColumns.push({ table, column });
      }
    },
    resetSelections(state) {
      state.selectedDb = '';
      state.selectedSchema = '';
      state.selectedTables = [];
      state.selectedColumns = [];
    },
  },
});

export const {
  setSelectedDb,
  setSelectedSchema,
  toggleTable,
  toggleColumn,
  resetSelections,
} = viewBuilderSlice.actions;

export default viewBuilderSlice.reducer;
