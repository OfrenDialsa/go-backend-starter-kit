package lib

import (
	"bytes"
	"strings"
	"text/template"
)

const (
	DefaultEmailSubject = "Verify Your Email"
	DefaultBody         = `
<div style="font-family: Arial, sans-serif; max-width:600px; margin:auto;">
	<h2>Email Verification</h2>

	<p>Hello <strong>{{.Name}}</strong>,</p>

	<p>Thank you for registering. Please verify your email by clicking the link below:</p>

	<p>
		<a href="{{.VerificationLink}}">
			Verify Email
		</a>
	</p>

	<p>If the link does not work, copy this URL into your browser:</p>

	<p>{{.VerificationLink}}</p>

	<hr>

	<p style="font-size:12px;color:#777;">
		If you did not create this account, you can ignore this email.<br>
		Sent by <strong>Ofren Dialsa</strong>
	</p>
</div>
`

	DefaultEmailSubjectResetPassword = "Reset Your Password"
	DefaultBodyResetPassword         = `
<div style="font-family: Arial, sans-serif; max-width:600px; margin:auto;">
	<h2>Password Reset</h2>

	<p>Hello <strong>{{.Name}}</strong>,</p>

	<p>A request was made to reset your password.</p>

	<p>
		<a href="{{.ResetLink}}">
			Reset Password
		</a>
	</p>

	<p>If the link does not work, copy this URL:</p>

	<p>{{.ResetLink}}</p>

	<hr>

	<p style="font-size:12px;color:#777;">
		If you did not request this, please ignore this email.<br>
		Sent by <strong>Ofren Dialsa</strong>
	</p>
</div>
`

	DefaultEmailSubjectPasswordResetSuccess = "Password Reset Successful"
	DefaultBodyPasswordResetSuccess         = `
<div style="font-family: Arial, sans-serif; max-width:600px; margin:auto;">
	<h2>Password Reset Successful</h2>

	<p>Hello <strong>{{.Name}}</strong>,</p>

	<p>Your password has been successfully reset.</p>

	<p>If this was not done by you, please contact support immediately.</p>

	<hr>

	<p style="font-size:12px;color:#777;">
		Security notification sent by <strong>Ofren Dialsa</strong>
	</p>
</div>
`

	DefaultEmailSubjectPasswordChangeSuccess = "Password Changed Successfully"
	DefaultBodyPasswordChangeSuccess         = `
<div style="font-family: Arial, sans-serif; max-width:600px; margin:auto;">
	<h2>Password Changed</h2>

	<p>Hello <strong>{{.Name}}</strong>,</p>

	<p>Your password has been changed successfully.</p>

	<p>If you did not make this change, please secure your account immediately.</p>

	<hr>

	<p style="font-size:12px;color:#777;">
		Security notification sent by <strong>Ofren Dialsa</strong>
	</p>
</div>
`
)

type EmailData struct {
	Name             string
	VerificationLink string
}

type EmailDataResetPassword struct {
	Name      string
	ResetLink string
}

type EmailDataPasswordResetSuccess struct {
	Name string
}

type EmailDataPasswordChangeSuccess struct {
	Name string
}

func BuildEmailBody(name, verificationLink string) (string, error) {
	htmlTemplate := DefaultBody

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := EmailData{
		Name:             name,
		VerificationLink: verificationLink,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func BuildEmailBodyResetPassword(name, resetPassword string) (string, error) {
	htmlTemplate := DefaultBodyResetPassword

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := EmailDataResetPassword{
		Name:      name,
		ResetLink: resetPassword,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func BuildEmailBodyPasswordResetSuccess(name string) (string, error) {
	htmlTemplate := DefaultBodyPasswordResetSuccess

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := EmailDataPasswordResetSuccess{
		Name: name,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func BuildEmailBodyPasswordChangeSuccess(name string) (string, error) {
	htmlTemplate := DefaultBodyPasswordChangeSuccess

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := EmailDataPasswordChangeSuccess{
		Name: name,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ExtractNameFromEmail(email string) string {
	if parts := strings.SplitN(email, "@", 2); len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return email
}
