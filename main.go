package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"mime/quotedprintable"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type Config struct {
	ImapServer string `json:"imap_server"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	ListenPort string `json:"listen_port"`
}

var config Config

func loadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return fmt.Errorf("failed to open config.json: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to decode config.json: %v", err)
	}

	return nil
}

func addMetadataBox(htmlContent string, msg *imapclient.FetchMessageBuffer) string {
	// Extract metadata
	var fromAddr string
	if len(msg.Envelope.From) > 0 {
		if msg.Envelope.From[0].Name != "" {
			fromAddr = msg.Envelope.From[0].Name + " <" + msg.Envelope.From[0].Mailbox + "@" + msg.Envelope.From[0].Host + ">"
		} else {
			fromAddr = msg.Envelope.From[0].Mailbox + "@" + msg.Envelope.From[0].Host
		}
	}

	subject := msg.Envelope.Subject
	date := msg.Envelope.Date.Format("2006-01-02 15:04:05")

	// Create metadata box HTML
	metadataBox := fmt.Sprintf(`
	<div style="
		background-color: #f5f5f5;
		border: 1px solid #ddd;
		border-radius: 5px;
		padding: 15px;
		margin: 10px;
		font-family: Arial, sans-serif;
		font-size: 14px;
		color: #333;
		box-shadow: 0 2px 4px rgba(0,0,0,0.1);
	">
		<div style="margin-bottom: 5px;"><strong>From:</strong> %s</div>
		<div style="margin-bottom: 5px;"><strong>Subject:</strong> %s</div>
		<div style="margin-bottom: 0;"><strong>Date:</strong> %s</div>
	</div>
	`, fromAddr, subject, date)

	// Find the body tag and insert metadata box after it
	bodyStart := strings.Index(htmlContent, "<body")
	if bodyStart != -1 {
		bodyTagEnd := strings.Index(htmlContent[bodyStart:], ">")
		if bodyTagEnd != -1 {
			insertPoint := bodyStart + bodyTagEnd + 1
			return htmlContent[:insertPoint] + metadataBox + htmlContent[insertPoint:]
		}
	}

	// If no body tag found, insert after html tag
	htmlStart := strings.Index(htmlContent, "<html")
	if htmlStart != -1 {
		htmlTagEnd := strings.Index(htmlContent[htmlStart:], ">")
		if htmlTagEnd != -1 {
			insertPoint := htmlStart + htmlTagEnd + 1
			return htmlContent[:insertPoint] + metadataBox + htmlContent[insertPoint:]
		}
	}

	// If no html or body tags, just prepend the metadata box
	return metadataBox + htmlContent
}

func getLatestEmail() (string, error) {
	options := &imapclient.Options{
		TLSConfig: &tls.Config{},
	}
	c, err := imapclient.DialTLS(config.ImapServer, options)
	if err != nil {
		return "", fmt.Errorf("failed to connect to IMAP server: %v", err)
	}
	defer c.Close()

	if err := c.Login(config.Email, config.Password).Wait(); err != nil {
		return "", fmt.Errorf("failed to login: %v", err)
	}

	mbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return "", fmt.Errorf("failed to select INBOX: %v", err)
	}

	if mbox.NumMessages == 0 {
		return "No messages in inbox", nil
	}

	seqSet := imap.SeqSetNum(mbox.NumMessages)
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	messages, err := c.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return "", fmt.Errorf("failed to fetch message: %v", err)
	}

	if len(messages) == 0 {
		return "No messages found", nil
	}

	msg := messages[0]
	var content strings.Builder

	content.WriteString("=== Latest Email ===\n\n")

	if len(msg.Envelope.From) > 0 {
		fromAddr := ""
		if msg.Envelope.From[0].Name != "" {
			fromAddr = msg.Envelope.From[0].Name + " <"
		}
		fromAddr += msg.Envelope.From[0].Mailbox + "@" + msg.Envelope.From[0].Host
		if msg.Envelope.From[0].Name != "" {
			fromAddr += ">"
		}
		content.WriteString(fmt.Sprintf("From: %s\n", fromAddr))
	}

	content.WriteString(fmt.Sprintf("Subject: %s\n", msg.Envelope.Subject))
	content.WriteString(fmt.Sprintf("Date: %s\n\n", msg.Envelope.Date.Format("2006-01-02 15:04:05")))

	// Extract and clean HTML content from body sections
	var rawContent string
	for _, body := range msg.BodySection {
		bodyStr := fmt.Sprintf("%s", body)
		rawContent += bodyStr
	}

	if rawContent != "" {
		// Find the actual HTML content by looking for DOCTYPE or <html> tag
		htmlStart := strings.Index(rawContent, "<!DOCTYPE")
		if htmlStart == -1 {
			htmlStart = strings.Index(rawContent, "<html")
		}

		if htmlStart != -1 {
			htmlContent := rawContent[htmlStart:]

			// Find the end of the HTML content (</html> tag)
			htmlEnd := strings.LastIndex(htmlContent, "</html>")
			if htmlEnd != -1 {
				htmlContent = htmlContent[:htmlEnd+7] // include </html>
			}

			// Decode quoted-printable encoding properly
			reader := quotedprintable.NewReader(strings.NewReader(htmlContent))
			decodedBytes := make([]byte, len(htmlContent)*2) // allocate extra space
			n, err := reader.Read(decodedBytes)

			var finalHTML string
			if err != nil && n == 0 {
				// If quoted-printable decoding fails, fall back to manual replacement
				htmlContent = regexp.MustCompile(`=\r?\n`).ReplaceAllString(htmlContent, "")
				htmlContent = strings.ReplaceAll(htmlContent, "=20", " ")
				htmlContent = strings.ReplaceAll(htmlContent, "=3D", "=")
				htmlContent = strings.ReplaceAll(htmlContent, "=E2=80=99", "'")
				htmlContent = strings.ReplaceAll(htmlContent, "=E2=80=93", "–")
				htmlContent = strings.ReplaceAll(htmlContent, "=E2=80=94", "—")
				finalHTML = htmlContent
			} else {
				decodedHTML := string(decodedBytes[:n])
				// Also clean up the decoded content
				htmlEnd = strings.LastIndex(decodedHTML, "</html>")
				if htmlEnd != -1 {
					decodedHTML = decodedHTML[:htmlEnd+7] // include </html>
				}
				finalHTML = decodedHTML
			}

			// Add metadata box at the top of the HTML
			return addMetadataBox(finalHTML, msg), nil
		}
	}

	// Fallback to text format if no HTML content found
	return content.String(), nil
}

func emailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	emailContent, err := getLatestEmail()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving email: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(emailContent))
}

func main() {
	err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	http.HandleFunc("/", emailHandler)

	log.Printf("Server starting on port %s", config.ListenPort)
	log.Fatal(http.ListenAndServe(":"+config.ListenPort, nil))
}
