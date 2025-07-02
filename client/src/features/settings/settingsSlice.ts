import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SettingsState {
  connectionString: string;
  savedConnections: string[];
  dataBaseInfo: any;
  connectionsMap: Record<string, string>;
}

const initialState: SettingsState = {
  connectionString: '',
  savedConnections: [],
  dataBaseInfo: null,
  connectionsMap: {},
};

const settingsSlice = createSlice({
  name: 'settings',
  initialState,
  reducers: {
    setConnectionString(state, action: PayloadAction<string>) {
      state.connectionString = action.payload;
    },
    setSavedConnections(state, action: PayloadAction<string[]>) {
      state.savedConnections = action.payload;
    },
    setDataForConnection(state, action: PayloadAction<any>) {
      state.dataBaseInfo = action.payload;
    },
    setConnectionsMap(state, action: PayloadAction<Record<string, string>>) {
      state.connectionsMap = action.payload;
    },
    appendTables(
      state,
      action: PayloadAction<{
        db: string;
        schema: string;
        tables: any[];
      }>,
    ) {
      const { db, schema, tables } = action.payload;
      const targetDb = state.dataBaseInfo?.find((d: any) => d.name === db);
      if (!targetDb) return;
      const targetSchema = targetDb.schemas?.find((s: any) => s.name === schema);
      if (!targetSchema) return;
      const existingNames = new Set(
        targetSchema.tables.map((t: any) => t.name),
      );
      tables.forEach((table) => {
        if (!existingNames.has(table.name)) {
          targetSchema.tables.push(table);
        }
      });
    },
  },
});

export const {
  setConnectionString,
  setSavedConnections,
  setDataForConnection,
  setConnectionsMap,
  appendTables,
} = settingsSlice.actions;

export default settingsSlice.reducer;
