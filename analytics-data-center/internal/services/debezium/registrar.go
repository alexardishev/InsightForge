package debezium

import (
	"analyticDataCenter/analytics-data-center/internal/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"analyticDataCenter/analytics-data-center/internal/logger"
)

func RegisterPostgresConnector(connectURL string, name string, pgConn string) error {
	u, err := url.Parse(pgConn)
	if err != nil {
		return fmt.Errorf("ошибка парсинга url: %w", err)
	}
	user := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	port := u.Port()
	dbname := u.Path[1:]

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("не верно указан порт: %w", err)
	}

	connector := DebeziumConnectorConfig{
		Name: fmt.Sprintf("conn_%s", name),
		Config: map[string]interface{}{
			"connector.class":                "io.debezium.connector.postgresql.PostgresConnector",
			"plugin.name":                    "pgoutput",
			"database.hostname":              host,
			"database.port":                  portInt,
			"database.user":                  user,
			"database.password":              password,
			"database.dbname":                dbname,
			"topic.prefix":                   fmt.Sprintf("dbserver_%s", name),
			"slot.name":                      fmt.Sprintf("slot_%s", name),
			"publication.name":               fmt.Sprintf("pub_%s", name),
			"tombstones.on.delete":           "false",
			"include.schema.changes":         "false",
			"decimal.handling.mode":          "double",
			"snapshot.mode":                  "never",
			"snapshot.new.tables":            "parallel",
			"key.converter":                  "org.apache.kafka.connect.json.JsonConverter",
			"value.converter":                "org.apache.kafka.connect.json.JsonConverter",
			"key.converter.schemas.enable":   "false",
			"value.converter.schemas.enable": "false",
		},
	}

	body, err := json.Marshal(connector)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/connectors", connectURL), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {

	}
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("неудачная попытка регистрации коннектора, status: %s, response: %s", resp.Status, string(bodyBytes))
	}

	return nil
}

func WaitConnectorsReady(connectURL string, conns []config.OLTPstorage, log *logger.Logger) {
	for _, c := range conns {
		name := c.Name
		maxWait := 30 * time.Second
		interval := 2 * time.Second
		timeout := time.After(maxWait)
		ticker := time.NewTicker(interval)

		log.Info("Ожидание коннекторов в статусе RUNNING", slog.String("name", name))

		for {
			select {
			case <-timeout:
				log.Warn("Коннектор не перешел в статус RUNNING", slog.String("name", name))
				ticker.Stop()
				goto nextConnector

			case <-ticker.C:
				statusURL := fmt.Sprintf("%s/connectors/%s/status", connectURL, name)
				resp, err := http.Get(statusURL)
				if err != nil {
					log.Warn("Неудачная попытка запроса коннектора", slog.String("name", name), slog.String("error", err.Error()))
					continue
				}
				var s struct {
					Connector struct {
						State string `json:"state"`
					} `json:"connector"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
					log.Warn("Невалидный ответ при проверке коннекторов", slog.String("name", name), slog.String("error", err.Error()))
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				if s.Connector.State == "RUNNING" {
					log.Info("Коннкетор в статусе RUNNING", slog.String("name", name))
					ticker.Stop()
					goto nextConnector
				}

				log.Info("Коннектор не в статусе RUNNING", slog.String("name", name), slog.String("state", s.Connector.State))
			}
		}
	nextConnector:
	}
}
