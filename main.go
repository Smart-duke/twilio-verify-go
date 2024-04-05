package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

var (
	signInTmpl = template.Must(template.New("signInForm").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Sign In</title>
		</head>
		<body>
			<form action="/send-otp" method="post">
				<label for="phone">Enter your phone number:</label>
				<input type="tel" id="phone" name="phone" required>
				<button type="submit">Send OTP</button>
			</form>
		</body>
		</html>
	`))

	verifyTmpl = template.Must(template.New("verifyForm").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Verify OTP</title>
		</head>
		<body>
			<form action="/verify-otp" method="post">
				<label for="code">Enter the OTP:</label>
				<input type="text" id="code" name="code" required>
				<input type="hidden" id="phone" name="phone" value="{{.}}">
				<button type="submit">Verify</button>
			</form>
		</body>
		</html>
	`))
)

func main() {
	// Twilio credentials
	accountSid := "TWILIO_ACCOUNT_SID"
	authToken := "TWILIO_AUTH_TOKEN"
	verifyServiceSid := "TWILIO_VERIFY_SERVICE_SID"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		signInTmpl.Execute(w, nil)
	})

	http.HandleFunc("/send-otp", func(w http.ResponseWriter, r *http.Request) {
		sendOTP(w, r, accountSid, authToken, verifyServiceSid)
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		verifyTmpl.Execute(w, r.URL.Query().Get("phone"))
	})

	http.HandleFunc("/verify-otp", func(w http.ResponseWriter, r *http.Request) {
		verifyOTP(w, r, accountSid, authToken, verifyServiceSid)
	})

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}

}

func sendOTP(w http.ResponseWriter, r *http.Request, accountSid, authToken, verifyServiceSid string) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	phoneNumber := r.Form.Get("phone")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &verify.CreateVerificationParams{}
	params.SetTo(phoneNumber)
	params.SetChannel("sms")

	_, err := client.VerifyV2.CreateVerification(verifyServiceSid, params)
	if err != nil {
		fmt.Fprintf(w, "Failed to send OTP: %v", err)
		return
	}

	// Redirect to verify page with phone number as a parameter (for simplicity in this example)
	http.Redirect(w, r, "/verify?phone="+phoneNumber, http.StatusSeeOther)
}

func verifyOTP(w http.ResponseWriter, r *http.Request, accountSid, authToken, verifyServiceSid string) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/verify", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	phoneNumber := r.Form.Get("phone")
	phoneNumber = "+" + phoneNumber
	code := r.Form.Get("code")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phoneNumber)
	params.SetCode(code)

	resp, err := client.VerifyV2.CreateVerificationCheck(verifyServiceSid, params)
	if err != nil {
		fmt.Fprintf(w, "Verification failed: %v", err)
		return
	}

	if *resp.Status == "approved" {
		fmt.Fprint(w, "Verification successful!")
	} else {
		fmt.Fprint(w, "Verification failed.")
	}
}
