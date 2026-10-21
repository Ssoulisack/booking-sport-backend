package mailer

import (
	"backend/internal/core/dto"
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type Config struct {
	Host     string
	Username string
	Password string
	Port     int
	From     string
}

type MailerRepository interface {
	SendMail(mailer dto.MailDTO) error
	SendInvitation(email string, tenant string, acceptURL string, inviter string, role string, receiverProfile string, tenantProfile string) error
}

type mailerRepository struct {
	mailer *gomail.Dialer
	config Config
}

// SendInvitation implements output.MailerRepository.
func (m *mailerRepository) SendInvitation(email, tenant, acceptURL, inviter, role, receiverProfile, tenantProfile string) error {
	logoURL := "https://fin.sabaiops.site/logo.png"
	companyName := "SabaiOps Co., Ltd."
	companyAddress := "123 Business Road, Tech District, 10110"
	companyEmail := "contact@sabaiops.com"
	baseURL := "https://fin.sabaiops.site"

	// User icon
	userIconHTML := ""
	if receiverProfile == "" {
		firstChar := string(email[0])
		userIconHTML = fmt.Sprintf(`<div style="width:60px;height:60px;border-radius:50%%;background:#0366d6;color:#fff;font-size:30px;font-weight:bold;text-align:center;line-height:60px;">%s</div>`, strings.ToUpper(firstChar))
	} else {
		userIconHTML = fmt.Sprintf(`<img src="%s" alt="Inviter" style="width:60px;height:60px;border-radius:4px;object-fit:cover;">`, receiverProfile)
	}

	// Tenant icon
	tenantIconHTML := ""
	if tenantProfile == "" {
		firstChar := "T"
		if len(tenant) > 0 {
			firstChar = string(tenant[0])
		}
		tenantIconHTML = fmt.Sprintf(`<div style="width:60px;height:60px;border-radius:50%%;background:#0366d6;color:#fff;font-size:30px;font-weight:bold;text-align:center;line-height:60px;">%s</div>`, strings.ToUpper(firstChar))
	} else {
		tenantIconHTML = fmt.Sprintf(`<img src="%s" alt="Tenant" style="width:60px;height:60px;border-radius:4px;object-fit:contain;">`, tenantProfile)
	}

	// Plain text fallback
	plainText := fmt.Sprintf(
		"%s invited you to collaborate on %s as a %s.\n\nAccept your invitation here: %s\n\nThis invitation will expire in 7 days.\n\nIf you were not expecting this email, please ignore it.\n\n--\n%s\n%s\nContact: %s",
		inviter, tenant, role, acceptURL, companyName, companyAddress, companyEmail,
	)

	// HTML version
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>You're Invited</title>
</head>
<body style="font-family: Arial, sans-serif; background: #f4f4f4; padding: 20px;">
  <div style="max-width: 600px; margin: auto; background: #ffffff; padding: 30px; border-radius: 8px;">
    <h2 style="text-align: center; color: #333;">Join the %s Workspace</h2>
    <div style="text-align: center; margin: 30px 0;">
      <table align="center" cellpadding="0" cellspacing="0" role="presentation">
        <tr>
          <td style="text-align: center;">%s</td>
          <td style="padding: 0 15px; color: #ccc; font-size: 24px;">+</td>
          <td style="text-align: center;">%s</td>
        </tr>
      </table>
    </div>
    <p style="text-align: center; font-size: 18px;">
      <strong>@%s</strong> has invited you to collaborate on <strong>%s</strong> as a <strong>%s</strong>.
    </p>
    <div style="text-align: center; margin: 50px;">
      <a href="%s" style="background: #0366d6; color: #ffffff; text-decoration: none; padding: 14px 26px; border-radius: 5px; font-weight: bold;">Join Now</a>
    </div>
    <p style="text-align: center; font-size: 14px;">
      This invitation link will expire in <strong>7 days</strong>. If you weren’t expecting this email, you can safely ignore it.
    </p>
    <p style="text-align: center; font-size: 13px;"><strong>Having trouble?</strong> Copy and paste the following link into your browser:</p>
    <p style="text-align: center; font-size: 13px; word-break: break-word;">
      <a href="%s">%s</a>
    </p>
    <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
    <div style="text-align: center; font-size: 12px; color: #888;">
      <p>
        <a href="%s/terms" style="color: #0366d6;">Terms</a> · 
        <a href="%s/privacy" style="color: #0366d6;">Privacy</a> · 
        <a href="%s" style="color: #0366d6;">Sign in</a>
      </p>
      <img src="%s" alt="Logo" style="max-width: 50px;" />
      <p><strong>%s</strong></p>
      <p>%s</p>
      <p><a href="mailto:%s">%s</a></p>
      <p style="margin-top: 10px;">&copy; %d %s. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`,
		tenant,
		userIconHTML,
		tenantIconHTML,
		inviter, tenant, role,
		acceptURL,
		acceptURL, acceptURL,

		baseURL, baseURL, baseURL,
		logoURL,
		companyName,
		companyAddress,
		companyEmail, companyEmail,
		time.Now().Year(), companyName,
	)

	subject := fmt.Sprintf("You're invited to collaborate on %s", tenant)

	go func() {
		_ = m.SendMail(dto.MailDTO{
			To:            email,
			Subject:       subject,
			Body:          plainText,
			BodyHTML:      html,
			UseHTMLLayout: true,
		})
	}()

	return nil
}

// SendMail handles sending via SMTP
func (m *mailerRepository) SendMail(mailer dto.MailDTO) error {
	message := gomail.NewMessage()

	// Set sender and recipient
	message.SetHeader("From", m.config.From)
	message.SetHeader("To", mailer.To)
	message.SetHeader("Subject", mailer.Subject)

	// Add headers to help spam filters
	message.SetHeader("X-Mailer", "GoMail")
	message.SetHeader("List-Unsubscribe", "<mailto:unsubscribe@example.com>")
	message.SetHeader("Precedence", "bulk") // ถ้าเป็นอีเมลแบบอัตโนมัติ
	message.SetHeader("MIME-Version", "1.0")

	// Add plain and HTML body
	plainBody := mailer.Body
	if plainBody == "" {
		plainBody = "To view this invitation, please use an HTML-compatible email client."
	}
	message.SetBody("text/plain", plainBody)

	if mailer.BodyHTML != "" {
		message.AddAlternative("text/html", mailer.BodyHTML)
	}

	// Send email
	if err := m.mailer.DialAndSend(message); err != nil {
		log.Printf("❌ Failed to send email: %v", err)
		return err
	}

	log.Printf("✅ Email sent to %s with subject: %s", mailer.To, mailer.Subject)
	return nil
}

func NewMailerRepository(config Config) MailerRepository {
	port := config.Port
	if port == 0 {
		port = 587
	}
	mailer := gomail.NewDialer(config.Host, port, config.Username, config.Password)
	return &mailerRepository{mailer: mailer, config: config}
}

// Ensure RedisRepository implements CacheRepository
// var _ output.MailerRepository = (*MailerRepository)(nil)
