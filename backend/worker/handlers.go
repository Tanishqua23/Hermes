package worker

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	gomail "gopkg.in/gomail.v2"
)

// HandlerFunc processes a job payload and returns a result map or an error.
type HandlerFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)

// Registry maps job type strings to their handler functions.
var Registry = map[string]HandlerFunc{
	"send_email":        handleSendEmail,
	"resize_image":      handleResizeImage,
	"send_notification": handleSendNotification,
	"generate_report":   handleGenerateReport,
	"process_payment":   handleProcessPayment,
}

// handleSendEmail sends a real email via Gmail SMTP.
// Required payload fields: "to", "subject", "body"
// Set GMAIL_USER and GMAIL_PASSWORD in your .env file.
func handleSendEmail(ctx context.Context, payload map[string]any) (map[string]any, error) {
	to, _ := payload["to"].(string)
	subject, _ := payload["subject"].(string)
	body, _ := payload["body"].(string)

	if to == "" {
		return nil, fmt.Errorf("send_email: missing 'to' in payload")
	}
	if subject == "" {
		return nil, fmt.Errorf("send_email: missing 'subject' in payload")
	}

	gmailUser := os.Getenv("GMAIL_USER")
	gmailPass := os.Getenv("GMAIL_PASSWORD")

	// Fall back to simulation if credentials not configured
	if gmailUser == "" || gmailPass == "" {
		simulateWork(ctx, 300*time.Millisecond, 700*time.Millisecond)
		return map[string]any{
			"status":     "simulated",
			"note":       "Set GMAIL_USER and GMAIL_PASSWORD in .env to send real emails",
			"to":         to,
			"subject":    subject,
			"sent_at":    time.Now().UTC(),
			"message_id": fmt.Sprintf("sim_msg_%d", time.Now().UnixNano()),
		}, nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", gmailUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	if body != "" {
		m.SetBody("text/plain", body)
	} else {
		m.SetBody("text/plain", "Sent via Hermes task queue.")
	}

	d := gomail.NewDialer("smtp.gmail.com", 587, gmailUser, gmailPass)

	if err := d.DialAndSend(m); err != nil {
		return nil, fmt.Errorf("send_email: smtp error: %w", err)
	}

	return map[string]any{
		"status":     "sent",
		"to":         to,
		"subject":    subject,
		"sent_at":    time.Now().UTC(),
		"message_id": fmt.Sprintf("msg_%d", time.Now().UnixNano()),
	}, nil
}

// handleResizeImage downloads an image from a URL, resizes it, and saves it locally.
// Required payload fields: "source_url", "width", "height"
// Output saved to backend/outputs/ folder.
func handleResizeImage(ctx context.Context, payload map[string]any) (map[string]any, error) {
	sourceURL, _ := payload["source_url"].(string)
	width := int(toFloat(payload["width"]))
	height := int(toFloat(payload["height"]))

	if sourceURL == "" {
		return nil, fmt.Errorf("resize_image: missing 'source_url' in payload")
	}
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	// Download the image with context timeout
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("resize_image: invalid URL: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resize_image: download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resize_image: download returned status %d", resp.StatusCode)
	}

	// Decode image
	img, err := imaging.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("resize_image: decode failed: %w", err)
	}

	origBounds := img.Bounds()

	// Resize using high-quality Lanczos resampling
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// Save to outputs folder next to the binary
	outDir := "outputs"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("resize_image: create output dir: %w", err)
	}

	filename := fmt.Sprintf("resized_%d_%dx%d.jpg", time.Now().UnixNano(), width, height)
	outPath := filepath.Join(outDir, filename)

	if err := imaging.Save(resized, outPath); err != nil {
		return nil, fmt.Errorf("resize_image: save failed: %w", err)
	}

	// Get file size
	info, _ := os.Stat(outPath)
	fileSize := int64(0)
	if info != nil {
		fileSize = info.Size()
	}

	return map[string]any{
		"status":          "resized",
		"original_url":    sourceURL,
		"original_width":  origBounds.Dx(),
		"original_height": origBounds.Dy(),
		"output_path":     outPath,
		"output_width":    width,
		"output_height":   height,
		"file_size_bytes": fileSize,
		"resized_at":      time.Now().UTC(),
	}, nil
}

// handleSendNotification sends a real Telegram message to your phone.
// Required payload fields: "user_id", "message"
// Set TELEGRAM_TOKEN and TELEGRAM_CHAT_ID in your .env file.
func handleSendNotification(ctx context.Context, payload map[string]any) (map[string]any, error) {
	userID, _ := payload["user_id"].(string)
	message, _ := payload["message"].(string)

	if userID == "" {
		return nil, fmt.Errorf("send_notification: missing 'user_id' in payload")
	}
	if message == "" {
		message = "Notification from Hermes"
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	// Fall back to simulation if credentials not configured
	if token == "" || chatID == "" {
		simulateWork(ctx, 100*time.Millisecond, 300*time.Millisecond)
		return map[string]any{
			"status":   "simulated",
			"note":     "Set TELEGRAM_TOKEN and TELEGRAM_CHAT_ID in .env to send real notifications",
			"user_id":  userID,
			"message":  message,
			"channel":  "telegram",
			"sent_at":  time.Now().UTC(),
		}, nil
	}

	text := fmt.Sprintf("🔔 *Hermes Notification*\n\n👤 User: `%s`\n💬 %s\n\n_Sent at %s_",
		userID, message, time.Now().Format("15:04:05"))

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	reqBody := fmt.Sprintf(`{"chat_id":"%s","text":%q,"parse_mode":"Markdown"}`, chatID, text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("send_notification: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send_notification: telegram API error: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("send_notification: telegram returned %d: %s", resp.StatusCode, string(respBody))
	}

	return map[string]any{
		"status":    "delivered",
		"user_id":   userID,
		"message":   message,
		"channel":   "telegram",
		"sent_at":   time.Now().UTC(),
	}, nil
}

// handleGenerateReport simulates a slow report generation job.
func handleGenerateReport(ctx context.Context, payload map[string]any) (map[string]any, error) {
	reportType, _ := payload["report_type"].(string)
	if reportType == "" {
		reportType = "default"
	}
	simulateWork(ctx, 1*time.Second, 3*time.Second)
	return map[string]any{
		"report_type":  reportType,
		"rows":         rand.Intn(10000) + 500,
		"url":          fmt.Sprintf("/reports/%s_%d.pdf", reportType, time.Now().UnixNano()),
		"generated_at": time.Now().UTC(),
	}, nil
}

// handleProcessPayment simulates payment processing.
// Randomly fails 20% of the time to demonstrate retry logic.
func handleProcessPayment(ctx context.Context, payload map[string]any) (map[string]any, error) {
	amount := toFloat(payload["amount"])
	currency, _ := payload["currency"].(string)

	if amount <= 0 {
		return nil, fmt.Errorf("process_payment: amount must be > 0")
	}
	if currency == "" {
		currency = "INR"
	}

	simulateWork(ctx, 300*time.Millisecond, 800*time.Millisecond)

	// 20% random failure — demonstrates retry logic
	if rand.Float64() < 0.2 {
		return nil, fmt.Errorf("process_payment: payment gateway timeout (transient error — will retry)")
	}

	return map[string]any{
		"transaction_id": fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		"amount":         amount,
		"currency":       currency,
		"status":         "captured",
		"processed_at":   time.Now().UTC(),
	}, nil
}

// simulateWork sleeps a random duration between min and max,
// respecting context cancellation.
func simulateWork(ctx context.Context, min, max time.Duration) {
	d := min + time.Duration(rand.Int63n(int64(max-min)))
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// toFloat safely converts interface{} to float64.
func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0
}