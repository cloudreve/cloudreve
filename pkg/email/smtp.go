package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/wneessen/go-mail"
)

// SMTPPool SMTP协议发送邮件
type SMTPPool struct {
	// Deprecated
	Config SMTPConfig

	config *setting.SMTP
	ch     chan *message
	chOpen bool
	l      logging.Logger
}

// SMTPConfig SMTP发送配置
type SMTPConfig struct {
	Name       string // 发送者名
	Address    string // 发送者地址
	ReplyTo    string // 回复地址
	Host       string // 服务器主机名
	Port       int    // 服务器端口
	User       string // 用户名
	Password   string // 密码
	Encryption bool   // 是否启用加密
	Keepalive  int    // SMTPPool 连接保留时长
}

type message struct {
	msg     *mail.Msg
	to      string
	subject string
	cid     string
	userID  int
}

// NewSMTPPool initializes a new SMTP based email sending queue.
func NewSMTPPool(config setting.Provider, logger logging.Logger) *SMTPPool {
	client := &SMTPPool{
		config: config.SMTP(context.Background()),
		ch:     make(chan *message, 30),
		chOpen: false,
		l:      logger,
	}

	client.Init()
	return client
}

// NewSMTPClient 新建SMTP发送队列
// Deprecated
func NewSMTPClient(config SMTPConfig) *SMTPPool {
	client := &SMTPPool{
		Config: config,
		ch:     make(chan *message, 30),
		chOpen: false,
	}

	client.Init()

	return client
}

// Send 发送邮件
func (client *SMTPPool) Send(ctx context.Context, to, title, body string) error {
	if !client.chOpen {
		return fmt.Errorf("SMTP pool is closed")
	}

	// 忽略通过QQ登录的邮箱
	if strings.HasSuffix(to, "@login.qq.com") {
		return nil
	}

	m := mail.NewMsg()
	if err := m.FromFormat(client.config.FromName, client.config.From); err != nil {
		return err
	}
	m.ReplyToFormat(client.config.FromName, client.config.ReplyTo)
	m.To(to)
	m.Subject(title)
	m.SetMessageID()
	m.SetBodyString(mail.TypeTextHTML, body)
	client.ch <- &message{
		msg:     m,
		subject: title,
		to:      to,
		cid:     logging.CorrelationID(ctx).String(),
		userID:  inventory.UserIDFromContext(ctx),
	}
	return nil
}

// Close 关闭发送队列
func (client *SMTPPool) Close() {
	if client.ch != nil {
		close(client.ch)
	}
}

// Init 初始化发送队列
func (client *SMTPPool) Init() {
	go func() {
		client.l.Info("Initializing and starting SMTP email pool...")
		defer func() {
			if err := recover(); err != nil {
				client.chOpen = false
				client.l.Error("Exception while sending email: %s, queue will be reset in 10 seconds.", err)
				time.Sleep(time.Duration(10) * time.Second)
				client.Init()
			}
		}()

		opts := []mail.Option{
			mail.WithPort(client.config.Port),
			mail.WithTimeout(time.Duration(client.config.Keepalive+5) * time.Second),
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithTLSPortPolicy(mail.TLSOpportunistic),
			mail.WithUsername(client.config.User), mail.WithPassword(client.config.Password),
		}
		if client.config.ForceEncryption {
			opts = append(opts, mail.WithSSL())
		}

		d, diaErr := mail.NewClient(client.config.Host, opts...)
		if diaErr != nil {
			client.l.Panic("Failed to create SMTP client: %s", diaErr)
			return
		}

		client.chOpen = true

		var err error
		open := false
		for {
			select {
			case m, ok := <-client.ch:
				if !ok {
					client.l.Info("Email queue closing...")
					client.chOpen = false
					return
				}

				if !open {
					if err = d.DialWithContext(context.Background()); err != nil {
						panic(err)
					}
					open = true
				}

				l := client.l.CopyWithPrefix(fmt.Sprintf("[Cid: %s]", m.cid))
				if err := d.Send(m.msg); err != nil {
					// Check if this is an SMTP RESET error after successful delivery
					var sendErr *mail.SendError
					var errParsed = errors.As(err, &sendErr)
					if errParsed && sendErr.Reason == mail.ErrSMTPReset {
						open = false
						l.Debug("SMTP RESET error, closing connection...")
						// https://github.com/wneessen/go-mail/issues/463
						continue // Don't treat this as a delivery failure since mail was sent
					}

					l.Warning("Failed to send email: %s, Cid=%s", err, m.cid)
				} else {
					l.Info("Email sent to %q, title: %q.", m.to, m.subject)
				}
			// 长时间没有新邮件，则关闭SMTP连接
			case <-time.After(time.Duration(client.config.Keepalive) * time.Second):
				if open {
					if err := d.Close(); err != nil {
						client.l.Warning("Failed to close SMTP connection: %s", err)
					}
					open = false
				}
			}
		}
	}()
}
