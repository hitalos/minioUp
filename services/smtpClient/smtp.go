package smtpClient

import (
	"context"
	"crypto/tls"
	"net"
	"net/smtp"
	"strconv"

	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/sendEmail"
)

func SendMail(ctx context.Context, to, subject string, tmpl config.TemplateString, params map[string]string, cfg config.SMTPConfig) error {
	tmpl.Params = params

	m := &sendEmail.Message{}
	m.SetFrom(cfg.From).
		SetTo(to).
		SetSubject(subject).
		SetHtml(tmpl.String())

	client, err := smtpConnect(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	return m.Send(client)
}

func smtpConnect(ctx context.Context, cfg config.SMTPConfig) (*smtp.Client, error) {
	var (
		hostport   = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
		tlsconfig  = &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
		smtpClient *smtp.Client
		err        error
	)

	if cfg.IsSecure {
		dialer := tls.Dialer{Config: tlsconfig}
		conn, err := dialer.DialContext(ctx, "tcp4", hostport)
		if err != nil {
			return nil, err
		}

		smtpClient, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return nil, err
		}
	} else {
		smtpClient, err = smtp.Dial(hostport)
		if err != nil {
			return nil, err
		}

		tlsconfig.InsecureSkipVerify = true
		if err := smtpClient.StartTLS(tlsconfig); err != nil {
			return nil, err
		}
	}

	if cfg.User != "" && cfg.Pass != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
		if err = smtpClient.Auth(auth); err != nil {
			return nil, err
		}
	}

	return smtpClient, nil
}
