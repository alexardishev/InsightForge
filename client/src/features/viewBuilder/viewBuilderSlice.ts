import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SelectedColumn {
  table: string;
  column: string;
}

interface JoinSide {
  source: string;
  schema: string;
  table: string;
  main_table: string;
  column_first: string;
  column_second: string;
}

interface Join {
  inner: JoinSide;
}

interface ViewBuilderState {
  selectedDb: string;
  selectedSchema: string;
  selectedTables: string[];
  selectedColumns: SelectedColumn[];
  joins: Join[];
  viewName: string;
  transformations: Record<string, Transform>;
}

export interface MappingJSON {
  mapping: Record<string, string>;
  type_field: string;
}

export interface Mapping {
  type_map?: string;
  mapping?: Record<string, string>;
  alias_new_column_transform?: string;
  type_field?: string;
  mapping_json?: MappingJSON[];
}

export interface Transform {
  type: string;
  mode: string;
  output_column: string;
  mapping: Mapping;
}

const initialState: ViewBuilderState = {
  selectedDb: '',
  selectedSchema: '',
  selectedTables: [],
  selectedColumns: [],
  joins: [],
  viewName: 'MyView',
  transformations: {},
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
    addJoin(state, action: PayloadAction<Join>) {
      state.joins.push(action.payload);
    },
    removeJoin(state, action: PayloadAction<number>) {
      state.joins.splice(action.payload, 1);
    },
    setTransformation(
      state,
      action: PayloadAction<{ table: string; column: string; transform: Transform | null }>,
    ) {
      const key = `${action.payload.table}.${action.payload.column}`;
      if (action.payload.transform) {
        state.transformations[key] = action.payload.transform;
      } else {
        delete state.transformations[key];
      }
    },
    setViewName(state, action: PayloadAction<string>) {
      state.viewName = action.payload;
    },
    resetSelections(state) {
      state.selectedDb = '';
      state.selectedSchema = '';
      state.selectedTables = [];
      state.selectedColumns = [];
      state.joins = [];
      state.viewName = 'MyView';
      state.transformations = {};
    },
  },
});

export const {
  setSelectedDb,
  setSelectedSchema,
  toggleTable,
  toggleColumn,
  addJoin,
  removeJoin,
  setTransformation,
  setViewName,
  resetSelections,
} = viewBuilderSlice.actions;

export default viewBuilderSlice.reducer;
