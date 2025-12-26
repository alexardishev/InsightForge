import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export interface SelectedColumn {
  db: string;
  schema: string;
  table: string;
  column: string;
  viewKey?: string;
  isUpdateKey?: boolean;
  alias?: string;
}

export interface TableSelection {
  selectedTables: string[];
  selectedColumns: SelectedColumn[];
}

export interface SourceSelection extends TableSelection {
  db: string;
  schema: string;
}

interface JoinSide {
  db: string;
  schema: string;
  table: string;
  column: string;
}

export interface JoinRule {
  type: 'INNER';
  left: JoinSide;
  right: JoinSide;
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

interface LoadState {
  loading: boolean;
  loaded?: boolean;
  error?: string;
}

interface TableLoadState extends LoadState {
  page: number;
  hasMore: boolean;
}

export interface ViewBuilderState {
  selectedDatabases: string[];
  selectedSchemasByDb: Record<string, string[]>;
  selectionsByDbSchema: Record<string, Record<string, TableSelection>>;
  schemaStatusByDb: Record<string, LoadState>;
  tableStatusByDbSchema: Record<string, Record<string, TableLoadState>>;
  joins: JoinRule[];
  viewName: string;
  transformations: Record<string, Transform>;
}

const initialState: ViewBuilderState = {
  selectedDatabases: [],
  selectedSchemasByDb: {},
  selectionsByDbSchema: {},
  schemaStatusByDb: {},
  tableStatusByDbSchema: {},
  joins: [],
  viewName: 'MyView',
  transformations: {},
};

export function flattenSelections(state: ViewBuilderState): SourceSelection[] {
  return Object.entries(state.selectionsByDbSchema).flatMap(([db, schemas]) =>
    Object.entries(schemas).map(([schema, selection]) => ({
      db,
      schema,
      selectedTables: selection.selectedTables,
      selectedColumns: selection.selectedColumns,
    })),
  );
}

const columnKey = (payload: {
  db: string;
  schema: string;
  table: string;
  column: string;
}) => `${payload.db}.${payload.schema}.${payload.table}.${payload.column}`;

const getSelection = (
  state: ViewBuilderState,
  db: string,
  schema: string,
): TableSelection => {
  if (!state.selectionsByDbSchema[db]) {
    state.selectionsByDbSchema[db] = {};
  }
  if (!state.selectionsByDbSchema[db][schema]) {
    state.selectionsByDbSchema[db][schema] = { selectedTables: [], selectedColumns: [] };
  }
  return state.selectionsByDbSchema[db][schema];
};

const pruneTransformations = (state: ViewBuilderState) => {
  const validKeys = new Set(
    flattenSelections(state).flatMap((selection) =>
      selection.selectedColumns.map((column) => columnKey(column)),
    ),
  );
  Object.keys(state.transformations).forEach((key) => {
    if (!validKeys.has(key)) {
      delete state.transformations[key];
    }
  });
};

const viewBuilderSlice = createSlice({
  name: 'viewBuilder',
  initialState,
  reducers: {
    setSelectedDatabases(state, action: PayloadAction<string[]>) {
      const nextDbs = new Set(action.payload);
      state.selectedDatabases = action.payload;

      Object.keys(state.selectedSchemasByDb).forEach((db) => {
        if (!nextDbs.has(db)) {
          delete state.selectedSchemasByDb[db];
        }
      });

      Object.keys(state.selectionsByDbSchema).forEach((db) => {
        if (!nextDbs.has(db)) {
          delete state.selectionsByDbSchema[db];
        }
      });

      Object.keys(state.schemaStatusByDb).forEach((db) => {
        if (!nextDbs.has(db)) {
          delete state.schemaStatusByDb[db];
        }
      });

      Object.keys(state.tableStatusByDbSchema).forEach((db) => {
        if (!nextDbs.has(db)) {
          delete state.tableStatusByDbSchema[db];
        }
      });

      state.joins = state.joins.filter(
        (join) => nextDbs.has(join.left.db) && nextDbs.has(join.right.db),
      );
      pruneTransformations(state);
    },
    setSchemasForDb(state, action: PayloadAction<{ db: string; schemas: string[] }>) {
      const { db, schemas } = action.payload;
      const allowed = new Set(schemas);
      state.selectedSchemasByDb[db] = schemas;

      if (state.selectionsByDbSchema[db]) {
        Object.keys(state.selectionsByDbSchema[db]).forEach((schema) => {
          if (!allowed.has(schema)) {
            delete state.selectionsByDbSchema[db][schema];
          }
        });
      }

      if (state.tableStatusByDbSchema[db]) {
        Object.keys(state.tableStatusByDbSchema[db]).forEach((schema) => {
          if (!allowed.has(schema)) {
            delete state.tableStatusByDbSchema[db][schema];
          }
        });
      }

      state.joins = state.joins.filter(
        (join) => !(join.left.db === db && !allowed.has(join.left.schema)) &&
          !(join.right.db === db && !allowed.has(join.right.schema)),
      );

      pruneTransformations(state);
    },
    toggleTable(
      state,
      action: PayloadAction<{ db: string; schema: string; table: string }>,
    ) {
      const { db, schema, table } = action.payload;
      const selection = getSelection(state, db, schema);
      if (selection.selectedTables.includes(table)) {
        selection.selectedTables = selection.selectedTables.filter((t) => t !== table);
        selection.selectedColumns = selection.selectedColumns.filter((c) => c.table !== table);
      } else {
        selection.selectedTables.push(table);
      }
      pruneTransformations(state);
    },
    toggleColumn(
      state,
      action: PayloadAction<
        SelectedColumn & { isPrimaryKey?: boolean; isUpdateKey?: boolean }
      >,
    ) {
      const { db, schema, table, column, isPrimaryKey, isUpdateKey } = action.payload;
      const selection = getSelection(state, db, schema);
      const idx = selection.selectedColumns.findIndex(
        (c) => c.table === table && c.column === column,
      );
      const key = columnKey({ db, schema, table, column });
      if (idx !== -1) {
        selection.selectedColumns.splice(idx, 1);
        delete state.transformations[key];
      } else {
        selection.selectedColumns.push({
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
      const selection = getSelection(state, db, schema);
      const otherTables = selection.selectedColumns.filter((c) => c.table !== table);
      const existingMap = new Map(
        selection.selectedColumns
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

      selection.selectedColumns = [...otherTables, ...updatedColumns];
      pruneTransformations(state);
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
      const selection = getSelection(state, db, schema);
      const col = selection.selectedColumns.find(
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
      const selection = getSelection(state, db, schema);
      const col = selection.selectedColumns.find(
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
      const selection = getSelection(state, db, schema);
      const col = selection.selectedColumns.find(
        (c) => c.table === table && c.column === column,
      );
      if (col) {
        col.alias = alias || undefined;
      }
    },
    addJoin(state, action: PayloadAction<JoinRule>) {
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
    setSchemaStatus(
      state,
      action: PayloadAction<{ db: string; status: Partial<LoadState> }>,
    ) {
      const { db, status } = action.payload;
      const prev = state.schemaStatusByDb[db] || { loading: false };
      state.schemaStatusByDb[db] = { ...prev, ...status };
    },
    setTableStatus(
      state,
      action: PayloadAction<{ db: string; schema: string; status: Partial<TableLoadState> }>,
    ) {
      const { db, schema, status } = action.payload;
      if (!state.tableStatusByDbSchema[db]) {
        state.tableStatusByDbSchema[db] = {};
      }
      const prev = state.tableStatusByDbSchema[db][schema] || {
        loading: false,
        page: 1,
        hasMore: true,
      };
      state.tableStatusByDbSchema[db][schema] = { ...prev, ...status };
    },
    resetSelections() {
      return initialState;
    },
  },
});

export const getColumnKey = columnKey;

export const {
  setSelectedDatabases,
  setSchemasForDb,
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
  setSchemaStatus,
  setTableStatus,
  resetSelections,
} = viewBuilderSlice.actions;

export default viewBuilderSlice.reducer;
