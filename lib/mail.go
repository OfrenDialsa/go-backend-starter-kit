package lib

import (
	"bytes"
	"strings"
	"text/template"
)

const (
	DefaultEmailSubject = "Verify Your Email"
	DefaultBody         = `
<div style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; max-width: 600px; margin: 0 auto; background-color: #ffffff; border: 1px solid #e0e0e0; border-radius: 8px; overflow: hidden;">
    <div style="background-color: #2563eb; padding: 20px; text-align: center;">
        <h2 style="color: #ffffff; margin: 0; font-size: 24px;">Email Verification</h2>
    </div>
    
    <div style="padding: 30px; line-height: 1.6; color: #333333;">
        <p style="font-size: 16px;">Hello <strong>{{.Name}}</strong>,</p>
        
        <p style="font-size: 16px;">Thank you for registering! Please verify your email address by clicking the button below to activate your account:</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationLink}}" 
               style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block; font-size: 16px;">
                Verify Email Address
            </a>
        </div>
        
        <p style="font-size: 14px; color: #666666;">
            If the button above does not work, please copy and paste the following link into your browser:
        </p>
        
        <div style="background-color: #f8f9fa; padding: 15px; border-radius: 4px; word-break: break-all; font-size: 13px; color: #2563eb; border: 1px solid #eeeeee;">
            {{.VerificationLink}}
        </div>
    </div>
    
    <div style="background-color: #f9fafb; padding: 20px; text-align: center; border-top: 1px solid #eeeeee;">
        <p style="font-size: 12px; color: #999999; margin: 0;">
            If you did not create this account, you can safely ignore this email.<br>
            Sent with by <strong>Ofren Dialsa</strong>
        </p>
    </div>
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
