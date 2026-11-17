package debezium

import (
	"analyticDataCenter/analytics-data-center/internal/config"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"analyticDataCenter/analytics-data-center/internal/lib/retention"
	"analyticDataCenter/analytics-data-center/internal/logger"
)

const (
	connectHTTPTimeout   = 10 * time.Second
	connectorWaitTimeout = 30 * time.Second
	connectorPollPeriod  = 2 * time.Second
)

// connectClient — HTTP-клиент для REST API Kafka Connect. Без таймаута
// зависший Connect подвесил бы старт приложения.
var connectClient = &http.Client{Timeout: connectHTTPTimeout}

// Имена объектов Debezium для источника — в одном месте: их используют
// регистрация коннектора и ожидание его готовности.
func ConnectorName(source string) string   { return "conn_" + source }
func SlotName(source string) string        { return "slot_" + source }
func PublicationName(source string) string { return "pub_" + source }
func TopicPrefix(source string) string     { return "dbserver_" + source }

// ConnectorOptions — настройки коннектора помимо строки подключения.
type ConnectorOptions struct {
	// TopicRetention — срок хранения топиков, которые создаёт Connect
	// (kafka.debezium_topic_retention): 0 — значение брокера по умолчанию,
	// отрицательный — бессрочно.
	TopicRetention time.Duration
	// HeartbeatInterval — как часто Debezium подтверждает позицию слота, даже если
	// событий нет (heartbeat.interval.ms); 0 — без heartbeat.
	HeartbeatInterval time.Duration
	// HeartbeatTable — таблица heartbeat источника ("table" или "schema.table"), куда
	// Debezium пишет на каждом heartbeat. Нужна, когда база-источник «тихая», а
	// соседние базы сервера пишут много: без своих изменений слот не двигается и
	// держит WAL всего сервера. Пусто — не писать. Таблицу создаёт EnsureHeartbeatTable.
	HeartbeatTable string
}

// RegisterPostgresConnector создаёт коннектор или обновляет его конфиг:
// PUT /connectors/<name>/config работает в обоих случаях. POST /connectors для
// уже существующего коннектора отвечает 409, и новые настройки до него не доходили.
// Если конфиг не изменился, PUT не отправляется: обновление конфига перезапускает
// коннектор, и CDC вставал бы на каждом старте приложения.
func RegisterPostgresConnector(connectURL string, name string, pgConn string, opts ConnectorOptions) error {
	connector := ConnectorName(name)
	connectorConfig, err := postgresConnectorConfig(name, pgConn, opts)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/connectors/%s/config", connectURL, url.PathEscape(connector))

	// Ошибка чтения текущего конфига не мешает регистрации: просто отправляем PUT.
	if current, found, err := currentConnectorConfig(endpoint); err == nil && found && maps.Equal(current, connectorConfig) {
		return nil
	}

	body, err := json.Marshal(connectorConfig)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := connectClient.Do(req)
	if err != nil {
		return fmt.Errorf("регистрация коннектора %s: %w", connector, err)
	}
	defer resp.Body.Close()
	// 201 — коннектор создан, 200 — конфиг существующего коннектора обновлён.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("неудачная попытка регистрации коннектора %s, status: %s, response: %s", connector, resp.Status, string(bodyBytes))
	}
	return nil
}

// currentConnectorConfig читает конфиг коннектора; found=false — коннектора нет (404).
func currentConnectorConfig(endpoint string) (map[string]string, bool, error) {
	resp, err := connectClient.Get(endpoint)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil // коннектора нет
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("status %s", resp.Status)
	}
	var cfg map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, false, err
	}
	return cfg, true, nil
}

// postgresConnectorConfig собирает конфиг коннектора из строки подключения.
// Значения — строки: REST API Connect принимает конфиг как map<string, string>.
// Ключ name совпадает с тем, что Connect возвращает в GET конфига, поэтому
// неизменённый конфиг сравнивается как равный.
func postgresConnectorConfig(name, pgConn string, opts ConnectorOptions) (map[string]string, error) {
	u, err := url.Parse(pgConn)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга url: %w", err)
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return nil, errors.New("в строке подключения не указана база данных")
	}
	port := u.Port()
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("не верно указан порт: %w", err)
	}
	password, _ := u.User.Password()

	connectorConfig := map[string]string{
		"name":                           ConnectorName(name),
		"connector.class":                "io.debezium.connector.postgresql.PostgresConnector",
		"plugin.name":                    "pgoutput",
		"database.hostname":              u.Hostname(),
		"database.port":                  port,
		"database.user":                  u.User.Username(),
		"database.password":              password,
		"database.dbname":                dbname,
		"topic.prefix":                   TopicPrefix(name),
		"slot.name":                      SlotName(name),
		"publication.name":               PublicationName(name),
		"tombstones.on.delete":           "false",
		"include.schema.changes":         "false",
		"decimal.handling.mode":          "double",
		"snapshot.mode":                  "never",
		"snapshot.new.tables":            "parallel",
		"key.converter":                  "org.apache.kafka.connect.json.JsonConverter",
		"value.converter":                "org.apache.kafka.connect.json.JsonConverter",
		"key.converter.schemas.enable":   "false",
		"value.converter.schemas.enable": "false",
	}
	if opts.HeartbeatInterval > 0 {
		connectorConfig["heartbeat.interval.ms"] = strconv.FormatInt(opts.HeartbeatInterval.Milliseconds(), 10)
	}
	if opts.HeartbeatTable != "" {
		if opts.HeartbeatInterval <= 0 {
			return nil, errors.New("heartbeat-таблица задана без heartbeat-интервала: Debezium выполняет запрос только на heartbeat")
		}
		quoted, err := quoteHeartbeatTable(opts.HeartbeatTable)
		if err != nil {
			return nil, err
		}
		connectorConfig["heartbeat.action.query"] = heartbeatActionQuery(quoted)
	}
	// Топики создаёт сам Connect с заданным сроком хранения (нужен topic.creation.enable=true
	// у воркера — так по умолчанию). Без replication.factor и partitions Connect топики не
	// создаёт, и их создавал брокер через auto-create со сроком хранения по умолчанию.
	if value, set := retention.Ms(opts.TopicRetention); set {
		connectorConfig["topic.creation.default.replication.factor"] = "-1" // значение брокера
		connectorConfig["topic.creation.default.partitions"] = "-1"         // значение брокера
		connectorConfig["topic.creation.default."+retention.ConfigKey] = value
	}
	return connectorConfig, nil
}

// WaitConnectorsReady ждёт, пока каждый коннектор перейдёт в RUNNING, но не дольше
// connectorWaitTimeout на коннектор. Первая проверка — сразу, без ожидания.
func WaitConnectorsReady(connectURL string, conns []config.OLTPstorage, log *logger.Logger) {
	for _, c := range conns {
		waitConnectorRunning(connectURL, ConnectorName(c.Name), log)
	}
}

func waitConnectorRunning(connectURL, connector string, log *logger.Logger) {
	log.Info("Ожидание коннекторов в статусе RUNNING", slog.String("name", connector))
	deadline := time.Now().Add(connectorWaitTimeout)
	for {
		state, err := connectorState(connectURL, connector)
		switch {
		case err != nil:
			log.Warn("Неудачная попытка запроса коннектора", slog.String("name", connector), slog.String("error", err.Error()))
		case state == "RUNNING":
			log.Info("Коннектор в статусе RUNNING", slog.String("name", connector))
			return
		default:
			log.Info("Коннектор не в статусе RUNNING", slog.String("name", connector), slog.String("state", state))
		}
		if time.Now().Add(connectorPollPeriod).After(deadline) {
			log.Warn("Коннектор не перешел в статус RUNNING", slog.String("name", connector))
			return
		}
		time.Sleep(connectorPollPeriod)
	}
}

// connectorState возвращает состояние коннектора. Ответ не 200 (например, 404 для
// несуществующего коннектора) — ошибка, а не пустое состояние.
func connectorState(connectURL, connector string) (string, error) {
	resp, err := connectClient.Get(fmt.Sprintf("%s/connectors/%s/status", connectURL, url.PathEscape(connector)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var s struct {
		Connector struct {
			State string `json:"state"`
		} `json:"connector"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", fmt.Errorf("невалидный ответ: %w", err)
	}
	return s.Connector.State, nil
}
