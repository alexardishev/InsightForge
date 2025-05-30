import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface SettingsState {
  connectionString: string;
  savedConnections: string[];
  dataBaseInfo: any;
}

const initialState: SettingsState = {
  connectionString: '',
  savedConnections: [],
  dataBaseInfo: null,
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
  },
});

export const {
  setConnectionString,
  setSavedConnections,
  setDataForConnection,
} = settingsSlice.actions;

export default settingsSlice.reducer;
