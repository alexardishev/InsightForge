import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SelectedColumn {
  table: string;
  column: string;
  viewKey?: string;
  isUpdateKey?: boolean;
  alias?: string;
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
    toggleColumn(
      state,
      action: PayloadAction<
        SelectedColumn & { isPrimaryKey?: boolean; isUpdateKey?: boolean }
      >,
    ) {
      const { table, column, isPrimaryKey, isUpdateKey } = action.payload;
      const idx = state.selectedColumns.findIndex(
        (c) => c.table === table && c.column === column,
      );
      if (idx !== -1) {
        state.selectedColumns.splice(idx, 1);
      } else {
        state.selectedColumns.push({
          table,
          column,
          isUpdateKey: isUpdateKey ?? isPrimaryKey ?? false,
        });
      }
    },
    setTableColumns(
      state,
      action: PayloadAction<{
        table: string;
        columns: { name: string; isPrimaryKey?: boolean; isUpdateKey?: boolean }[];
      }>,
    ) {
      const { table, columns } = action.payload;
      const otherTables = state.selectedColumns.filter((c) => c.table !== table);
      const existingMap = new Map(
        state.selectedColumns
          .filter((c) => c.table === table)
          .map((c) => [c.column, c]),
      );

      const updatedColumns = columns.map((col) => {
        const existing = existingMap.get(col.name);
        const isUpdate =
          existing?.isUpdateKey ?? col.isUpdateKey ?? col.isPrimaryKey ?? false;

        return {
          table,
          column: col.name,
          viewKey: existing?.viewKey,
          isUpdateKey: isUpdate,
          alias: existing?.alias,
        } as SelectedColumn;
      });

      state.selectedColumns = [...otherTables, ...updatedColumns];
    },
    setViewKey(
      state,
      action: PayloadAction<{ table: string; column: string; viewKey: string }>,
    ) {
      const { table, column, viewKey } = action.payload;
      const col = state.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.viewKey = viewKey || undefined;
      }
    },
    setUpdateKey(
      state,
      action: PayloadAction<{
        table: string;
        column: string;
        isUpdateKey: boolean;
      }>,
    ) {
      const { table, column, isUpdateKey } = action.payload;
      const col = state.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.isUpdateKey = isUpdateKey;
      }
    },
    setColumnAlias(
      state,
      action: PayloadAction<{ table: string; column: string; alias: string }>,
    ) {
      const { table, column, alias } = action.payload;
      const col = state.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.alias = alias || undefined;
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
  setTableColumns,
  setViewKey,
  setUpdateKey,
  setColumnAlias,
  addJoin,
  removeJoin,
  setTransformation,
  setViewName,
  resetSelections,
} = viewBuilderSlice.actions;

export default viewBuilderSlice.reducer;
