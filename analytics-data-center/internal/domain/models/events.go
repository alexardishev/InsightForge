package models

type CDCEvent struct {
	Event string       `json:"event"`
	ID    string       `json:"id"`
	Data  CDCEventData `json:"data"`
}

type CDCEventData struct {
	Before      map[string]interface{} `json:"before"`
	After       map[string]interface{} `json:"after"`
	Source      CDCSource              `json:"source"`
	Op          string                 `json:"op"`          // "c" - create, "u" - update, "d" - delete, "r" - snapshot
	Transaction interface{}            `json:"transaction"` // можешь позже типизировать
	TsMs        int64                  `json:"ts_ms"`
	TsUs        int64                  `json:"ts_us"`
	TsNs        int64                  `json:"ts_ns"`
}

type CDCSource struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	TsMs      int64  `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	DB        string `json:"db"`
	Sequence  string `json:"sequence"`
	TsUs      int64  `json:"ts_us"`
	TsNs      int64  `json:"ts_ns"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	TxID      int64  `json:"txId"`
	LSN       int64  `json:"lsn"`
	Xmin      *int64 `json:"xmin"` // может быть null
}

type Event struct {
	EventName string
	EventData map[string]interface{} // может быть любым типом данных
}
