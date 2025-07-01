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
  },
});

export const {
  setConnectionString,
  setSavedConnections,
  setDataForConnection,
  setConnectionsMap,
} = settingsSlice.actions;

export default settingsSlice.reducer;
