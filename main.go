package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

var (
	store      = sessions.NewCookieStore([]byte("secret")) // Change "secret" to your desired secret key
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
				<button type="submit">Verify</button>
			</form>
		</body>
		</html>
	`))
)

func main() {
	godotenv.Load()
	// Twilio credentials
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	verifyServiceSid := os.Getenv("TWILIO_VERIFY_SERVICE_SID")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		signInTmpl.Execute(w, nil)
	})

	http.HandleFunc("/send-otp", func(w http.ResponseWriter, r *http.Request) {
		sendOTP(w, r, accountSid, authToken, verifyServiceSid)
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		phoneNumber := session.Values["phone"]
		verifyTmpl.Execute(w, phoneNumber)
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

	// Parse form data
	r.ParseForm()
	phoneNumber := r.Form.Get("phone")

	if !strings.HasPrefix(phoneNumber, "+") {
		countryCode := "your_country_code" // Replace with the appropriate default country code
		phoneNumber = "+" + countryCode + phoneNumber
	}

	// Save phone number in session
	session, _ := store.Get(r, "session-name")
	session.Values["phone"] = phoneNumber
	session.Save(r, w)

	// Create Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	// Set verification parameters
	params := &verify.CreateVerificationParams{}
	params.SetTo(phoneNumber)
	params.SetChannel("sms")

	// Create verification
	_, err := client.VerifyV2.CreateVerification(verifyServiceSid, params)
	if err != nil {
		fmt.Fprintf(w, "Failed to send OTP: %v", err)
		return
	}

	// Redirect to verify page
	http.Redirect(w, r, "/verify", http.StatusSeeOther)
}

func verifyOTP(w http.ResponseWriter, r *http.Request, accountSid, authToken, verifyServiceSid string) {
	// Check if the request method is POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/verify", http.StatusSeeOther)
		return
	}

	// Retrieve the session and the stored phone number
	session, _ := store.Get(r, "session-name")
	phoneNumber := "+" + session.Values["phone"].(string)

	// Retrieve the code from the form data
	r.ParseForm()
	code := r.Form.Get("code")

	// Check if the code is empty or null
	if code == "" {
		fmt.Fprintf(w, "Verification failed: No code provided.")
		return
	}

	// Create the Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	// Create the verification check parameters
	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phoneNumber)
	params.SetCode(code)

	// Perform the verification check
	resp, err := client.VerifyV2.CreateVerificationCheck(verifyServiceSid, params)
	if err != nil {
		fmt.Fprintf(w, "Verification failed: %v", err)
		return
	}

	// Check the verification status
	if resp != nil && resp.Status != nil && *resp.Status == "approved" {
		fmt.Fprint(w, "Verification successful!")
	} else {
		fmt.Fprint(w, "Verification failed.")
	}
}
