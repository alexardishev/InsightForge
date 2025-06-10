package smtpsender

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"fmt"
	"log/slog"
	"net/smtp"
)

type SMTP struct {
	Host           string `json:"host,omitempty"`
	Port           int    `json:"port,omitempty"`
	UserName       string `json:"user_name,omitempty"`
	Password       string `json:"password,omitempty"`
	AdminEmail     string `json:"admin_email,omitempty"`
	FromEmail      string `json:"from_email,omitempty"`
	EventQueueSMTP chan models.Event
	log            *loggerpkg.Logger
}

func NewSMTP(
	Host string,
	Port int,
	UserName string,
	Password string,
	AdminEmail string,
	FromEmail string,
	Logger *loggerpkg.Logger,
) *SMTP {
	smtp := &SMTP{
		Host:           Host,
		Port:           Port,
		UserName:       UserName,
		Password:       Password,
		AdminEmail:     AdminEmail,
		FromEmail:      FromEmail,
		EventQueueSMTP: make(chan models.Event),
		log:            Logger,
	}
	go smtp.emailSendWorker()
	return smtp
}

func (s *SMTP) emailSendWorker() {
	for event := range s.EventQueueSMTP {
		log := s.log.With(
			slog.String("component", "EmailSendWorker"),
		)
		log.InfoMsg(loggerpkg.MsgEventWorkerReceived)
		// ctx := context.Background()
		err := s.SendEmail(event)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgEventWorkerError, slog.String("error", err.Error()))
		}
	}
}

func (s *SMTP) SendEmail(event models.Event) error {
	switch event.EventName {
	case "TableChanged":
		s.prepareMail(event)
		return nil
	}
	return nil
}
func (s *SMTP) prepareMail(event models.Event) error {
	auth := smtp.PlainAuth("", s.UserName, s.Password, s.Host)

	value, ok := event.EventData["messageEmail"]
	if !ok {
		return fmt.Errorf("messageEmail not found in EventData")
	}

	body, ok := value.(string)
	if !ok {
		return fmt.Errorf("messageEmail is not a string")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Уведомление\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		s.FromEmail, s.AdminEmail, body)

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	err := smtp.SendMail(addr, auth, s.FromEmail, []string{s.AdminEmail}, []byte(msg))
	if err != nil {
		s.log.ErrorMsg(loggerpkg.MsgEmailSendFailed, slog.String("error", err.Error()))
		return err
	}
	return nil
}
