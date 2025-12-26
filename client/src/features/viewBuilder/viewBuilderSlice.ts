import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SelectedColumn {
  db: string;
  schema: string;
  table: string;
  column: string;
  viewKey?: string;
  isUpdateKey?: boolean;
  alias?: string;
}

interface SourceSelection {
  db: string;
  schema: string;
  selectedTables: string[];
  selectedColumns: SelectedColumn[];
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
  currentDb: string;
  currentSchema: string;
  selectedSources: SourceSelection[];
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
  currentDb: '',
  currentSchema: '',
  selectedSources: [],
  joins: [],
  viewName: 'MyView',
  transformations: {},
};

const columnKey = (payload: {
  db: string;
  schema: string;
  table: string;
  column: string;
}) => `${payload.db}.${payload.schema}.${payload.table}.${payload.column}`;

const findOrCreateSource = (
  state: ViewBuilderState,
  db: string,
  schema: string,
): SourceSelection => {
  let source = state.selectedSources.find(
    (s) => s.db === db && s.schema === schema,
  );
  if (!source) {
    source = { db, schema, selectedTables: [], selectedColumns: [] };
    state.selectedSources.push(source);
  }
  return source;
};

const cleanupEmptySources = (state: ViewBuilderState) => {
  state.selectedSources = state.selectedSources.filter(
    (source) => source.selectedTables.length > 0 || source.selectedColumns.length > 0,
  );
};

const viewBuilderSlice = createSlice({
  name: 'viewBuilder',
  initialState,
  reducers: {
    setCurrentDb(state, action: PayloadAction<string>) {
      state.currentDb = action.payload;
      state.currentSchema = '';
    },
    setCurrentSchema(state, action: PayloadAction<string>) {
      state.currentSchema = action.payload;
    },
    toggleTable(
      state,
      action: PayloadAction<{ db: string; schema: string; table: string }>,
    ) {
      const { db, schema, table } = action.payload;
      const source = findOrCreateSource(state, db, schema);
      if (source.selectedTables.includes(table)) {
        source.selectedTables = source.selectedTables.filter((t) => t !== table);
        source.selectedColumns = source.selectedColumns.filter(
          (c) => !(c.table === table && c.db === db && c.schema === schema),
        );
      } else {
        source.selectedTables.push(table);
      }
      cleanupEmptySources(state);
    },
    toggleColumn(
      state,
      action: PayloadAction<
        SelectedColumn & { isPrimaryKey?: boolean; isUpdateKey?: boolean }
      >,
    ) {
      const { db, schema, table, column, isPrimaryKey, isUpdateKey } =
        action.payload;
      const source = findOrCreateSource(state, db, schema);
      const idx = source.selectedColumns.findIndex(
        (c) => c.table === table && c.column === column,
      );
      const key = columnKey({ db, schema, table, column });
      if (idx !== -1) {
        source.selectedColumns.splice(idx, 1);
        delete state.transformations[key];
      } else {
        source.selectedColumns.push({
          db,
          schema,
          table,
          column,
          isUpdateKey: isUpdateKey ?? isPrimaryKey ?? false,
        });
      }
    },
    setTableColumns(
      state,
      action: PayloadAction<{
        db: string;
        schema: string;
        table: string;
        columns: { name: string; isPrimaryKey?: boolean; isUpdateKey?: boolean }[];
      }>,
    ) {
      const { db, schema, table, columns } = action.payload;
      const source = findOrCreateSource(state, db, schema);
      const otherTables = source.selectedColumns.filter((c) => c.table !== table);
      const existingMap = new Map(
        source.selectedColumns
          .filter((c) => c.table === table)
          .map((c) => [c.column, c]),
      );

      const updatedColumns = columns.map((col) => {
        const existing = existingMap.get(col.name);
        const isUpdate =
          existing?.isUpdateKey ?? col.isUpdateKey ?? col.isPrimaryKey ?? false;

        return {
          db,
          schema,
          table,
          column: col.name,
          viewKey: existing?.viewKey,
          isUpdateKey: isUpdate,
          alias: existing?.alias,
        } as SelectedColumn;
      });

      source.selectedColumns = [...otherTables, ...updatedColumns];
      if (columns.length === 0) {
        cleanupEmptySources(state);
      }
    },
    setViewKey(
      state,
      action: PayloadAction<{
        db: string;
        schema: string;
        table: string;
        column: string;
        viewKey: string;
      }>,
    ) {
      const { db, schema, table, column, viewKey } = action.payload;
      const source = findOrCreateSource(state, db, schema);
      const col = source.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.viewKey = viewKey || undefined;
      }
    },
    setUpdateKey(
      state,
      action: PayloadAction<{
        db: string;
        schema: string;
        table: string;
        column: string;
        isUpdateKey: boolean;
      }>,
    ) {
      const { db, schema, table, column, isUpdateKey } = action.payload;
      const source = findOrCreateSource(state, db, schema);
      const col = source.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.isUpdateKey = isUpdateKey;
      }
    },
    setColumnAlias(
      state,
      action: PayloadAction<{
        db: string;
        schema: string;
        table: string;
        column: string;
        alias: string;
      }>,
    ) {
      const { db, schema, table, column, alias } = action.payload;
      const source = findOrCreateSource(state, db, schema);
      const col = source.selectedColumns.find(
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
      action: PayloadAction<{
        db: string;
        schema: string;
        table: string;
        column: string;
        transform: Transform | null;
      }>,
    ) {
      const key = columnKey(action.payload);
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
      state.currentDb = '';
      state.currentSchema = '';
      state.selectedSources = [];
      state.joins = [];
      state.viewName = 'MyView';
      state.transformations = {};
    },
  },
});

export const {
  setCurrentDb,
  setCurrentSchema,
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

export const getColumnKey = columnKey;

export default viewBuilderSlice.reducer;
