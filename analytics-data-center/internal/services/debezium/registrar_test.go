package debezium

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"analyticDataCenter/analytics-data-center/internal/config"
	"analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/stretchr/testify/require"
)

type recordedRequest struct {
	method string
	path   string
	body   map[string]string
}

// fakeConnect — REST API Kafka Connect с хранилищем конфигов: GET
// /connectors/<name>/config отдаёт сохранённый конфиг или 404, PUT сохраняет
// конфиг (если putStatus успешный) и отвечает putStatus.
type fakeConnect struct {
	*httptest.Server
	mu        sync.Mutex
	configs   map[string]map[string]string
	requests  []recordedRequest
	putStatus int
}

func newFakeConnect(t *testing.T, putStatus int) *fakeConnect {
	t.Helper()
	f := &fakeConnect{configs: make(map[string]map[string]string), putStatus: putStatus}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeConnect) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	var body map[string]string
	_ = json.Unmarshal(raw, &body)
	f.requests = append(f.requests, recordedRequest{method: r.Method, path: r.URL.Path, body: body})

	connector := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/connectors/"), "/config")
	switch r.Method {
	case http.MethodGet:
		cfg, ok := f.configs[connector]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":404,"message":"Connector not found"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(cfg)
	case http.MethodPut:
		if f.putStatus == http.StatusOK || f.putStatus == http.StatusCreated {
			f.configs[connector] = body
		}
		w.WriteHeader(f.putStatus)
		_, _ = w.Write([]byte(`{}`))
	}
}

func (f *fakeConnect) puts() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var puts []recordedRequest
	for _, r := range f.requests {
		if r.method == http.MethodPut {
			puts = append(puts, r)
		}
	}
	return puts
}

func testLogger() *logger.Logger {
	return &logger.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

const testConn = "postgresql://debezium:secret@pg:5432/shop"

func TestRegisterPostgresConnectorCreatesConnectorWithPut(t *testing.T) {
	connect := newFakeConnect(t, http.StatusCreated)

	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}))

	puts := connect.puts()
	require.Len(t, puts, 1)
	require.Equal(t, "/connectors/conn_db1/config", puts[0].path,
		"PUT creates the connector or updates its config; POST fails with 409 for an existing connector")
	body := puts[0].body
	require.Equal(t, "conn_db1", body["name"])
	require.Equal(t, "slot_db1", body["slot.name"], "PUT body is the config itself, not {name, config}")
	require.Equal(t, "pub_db1", body["publication.name"])
	require.Equal(t, "dbserver_db1", body["topic.prefix"])
	require.Equal(t, "shop", body["database.dbname"])
	require.Equal(t, "5432", body["database.port"], "Connect config values are strings")
}

func TestRegisterPostgresConnectorSkipsUnchangedConfig(t *testing.T) {
	connect := newFakeConnect(t, http.StatusOK)
	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}))

	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}))

	require.Len(t, connect.puts(), 1, "an unchanged config is not re-sent: Connect would restart the connector on every app start")
}

func TestRegisterPostgresConnectorUpdatesChangedConfig(t *testing.T) {
	connect := newFakeConnect(t, http.StatusOK)
	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}))

	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", "postgresql://debezium:rotated@pg:5432/shop", ConnectorOptions{}))

	puts := connect.puts()
	require.Len(t, puts, 2)
	require.Equal(t, "rotated", puts[1].body["database.password"])
}

func TestRegisterPostgresConnectorReportsHTTPError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusConflict, http.StatusInternalServerError} {
		connect := newFakeConnect(t, status)
		require.Error(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}), status)
	}
}

func TestRegisterPostgresConnectorRejectsInvalidConnectionString(t *testing.T) {
	connect := newFakeConnect(t, http.StatusOK)
	for _, conn := range []string{
		"postgresql://debezium:secret@pg:5432",  // нет имени базы
		"postgresql://debezium:secret@pg:5432/", // пустое имя базы
		"postgresql://debezium:secret@pg/shop",  // нет порта
	} {
		require.Error(t, RegisterPostgresConnector(connect.URL, "db1", conn, ConnectorOptions{}), conn)
	}
	require.Empty(t, connect.requests, "an invalid connection string must not reach Connect")
}

func TestWaitConnectorsReadyChecksConnectorByItsName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connectors/conn_db1/status" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"connector":{"state":"RUNNING"}}`))
	}))
	defer srv.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		WaitConnectorsReady(srv.URL, []config.OLTPstorage{{Name: "db1"}}, testLogger())
	}()
	select {
	case <-done:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("a RUNNING connector must be detected right away, not after the 30 s timeout")
	}
}

func TestConnectorNames(t *testing.T) {
	require.Equal(t, "conn_db1", ConnectorName("db1"))
	require.Equal(t, "slot_db1", SlotName("db1"))
	require.Equal(t, "pub_db1", PublicationName("db1"))
	require.Equal(t, "dbserver_db1", TopicPrefix("db1"))
}

func TestConnectorConfigCreatesTopicsWithRetention(t *testing.T) {
	cfg, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{TopicRetention: 336 * time.Hour})
	require.NoError(t, err)

	// Без replication.factor и partitions Connect топики не создаёт — их создаёт брокер
	// через auto-create со сроком хранения по умолчанию.
	require.Equal(t, "-1", cfg["topic.creation.default.replication.factor"])
	require.Equal(t, "-1", cfg["topic.creation.default.partitions"])
	require.Equal(t, "1209600000", cfg["topic.creation.default.retention.ms"])
}

func TestConnectorConfigWithoutRetentionLeavesTopicsToBroker(t *testing.T) {
	cfg, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{})
	require.NoError(t, err)

	for key := range cfg {
		require.False(t, strings.HasPrefix(key, "topic.creation."), key)
	}
}

func TestRegisterPostgresConnectorAppliesChangedRetention(t *testing.T) {
	connect := newFakeConnect(t, http.StatusOK)
	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{}))

	require.NoError(t, RegisterPostgresConnector(connect.URL, "db1", testConn, ConnectorOptions{TopicRetention: 336 * time.Hour}))

	puts := connect.puts()
	require.Len(t, puts, 2, "a changed retention must reach the already registered connector")
	require.Equal(t, "1209600000", puts[1].body["topic.creation.default.retention.ms"])
}
