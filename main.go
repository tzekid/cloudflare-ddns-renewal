package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// getEnv fetches a required environment variable and logs a fatal error if not found.
func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Environment variable %q not set", key)
	}
	return val
}

func sendTelegramMessage(message string) {
	// Fetch Telegram credentials from environment variables.
	telegramBotToken := getEnv("TELEGRAM_BOT_TOKEN")
	telegramChatID := getEnv("TELEGRAM_CHAT_ID")

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	data := url.Values{}
	data.Set("chat_id", telegramChatID)
	data.Set("text", message)

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("Error sending telegram message: %v", err)
		return
	}
	defer resp.Body.Close()
	// Optionally, you can check resp.StatusCode or inspect the response body.
}

// updateDomain updates the root A record and its wildcard for a single domain.
func updateDomain(ctx context.Context, api *cloudflare.API, currentIP, domain string) error {
	start := time.Now()
	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for %s: %w", domain, err)
	}
	rc := &cloudflare.ResourceContainer{Identifier: zoneID, Type: "zone"}

	rootName := domain
	wildcardName := "*." + domain

	// Fetch existing root record
	rootRecords, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{Type: "A", Name: rootName})
	if err != nil {
		return fmt.Errorf("error listing root record for %s: %w", domain, err)
	}
	var rootRec *cloudflare.DNSRecord
	if len(rootRecords) > 0 {
		rootRec = &rootRecords[0]
	}

	// Fetch existing wildcard record
	wildcardRecords, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{Type: "A", Name: wildcardName})
	if err != nil {
		return fmt.Errorf("error listing wildcard record for %s: %w", domain, err)
	}
	var wildcardRec *cloudflare.DNSRecord
	if len(wildcardRecords) > 0 {
		wildcardRec = &wildcardRecords[0]
	}

	if rootRec == nil {
		return fmt.Errorf("A record for %s not found", rootName)
	}
	if wildcardRec == nil {
		return fmt.Errorf("A record for %s not found", wildcardName)
	}

	// Update root record if needed
	if rootRec.Content == currentIP {
		fmt.Printf("%s already up to date (%s)\n", rootName, currentIP)
	} else {
		fmt.Printf("Updating %s from %s to %s\n", rootName, rootRec.Content, currentIP)
		_, err = api.UpdateDNSRecord(ctx, rc, cloudflare.UpdateDNSRecordParams{
			ID:      rootRec.ID,
			Type:    "A",
			Name:    rootName,
			Content: currentIP,
			TTL:     rootRec.TTL,
			Proxied: rootRec.Proxied,
		})
		if err != nil {
			return fmt.Errorf("failed updating %s: %w", rootName, err)
		}
		sendTelegramMessage(fmt.Sprintf("IP for %s updated to %s", rootName, currentIP))
	}

	// Update wildcard if needed
	if wildcardRec.Content == currentIP {
		fmt.Printf("%s already up to date (%s)\n", wildcardName, currentIP)
	} else {
		fmt.Printf("Updating %s from %s to %s\n", wildcardName, wildcardRec.Content, currentIP)
		_, err = api.UpdateDNSRecord(ctx, rc, cloudflare.UpdateDNSRecordParams{
			ID:      wildcardRec.ID,
			Type:    "A",
			Name:    wildcardName,
			Content: currentIP,
			TTL:     wildcardRec.TTL,
			Proxied: wildcardRec.Proxied,
		})
		if err != nil {
			return fmt.Errorf("failed updating %s: %w", wildcardName, err)
		}
		sendTelegramMessage(fmt.Sprintf("IP for %s updated to %s", wildcardName, currentIP))
	}

	fmt.Printf("Finished %s in %s\n", domain, time.Since(start).Truncate(time.Millisecond))
	return nil
}

func parseDomains() []string {
	// DOMAINS can be a comma or space separated list. Falls back to DOMAIN or legacy default.
	domainsVar := os.Getenv("DOMAINS")
	if domainsVar == "" {
		single := os.Getenv("DOMAIN")
		if single != "" {
			return []string{strings.TrimSpace(single)}
		}
		// Legacy default
		return []string{"plosca.ru"}
	}
	// Replace common separators with comma then split
	cleaned := strings.NewReplacer("\n", ",", " ", ",", ";", ",", "|", ",").Replace(domainsVar)
	parts := strings.Split(cleaned, ",")
	var domains []string
	for _, p := range parts {
		d := strings.TrimSpace(p)
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		domains = []string{"plosca.ru"}
	}
	return domains
}

func main() {
	// 1. Determine current public IP.
	resp, err := http.Get("http://ipinfo.io/ip")
	if err != nil {
		log.Fatalf("Failed to get current IP: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read IP response: %v", err)
	}
	currentIP := strings.TrimSpace(string(body))
	fmt.Printf("Current IP: %s\n", currentIP)

	// 2. Cloudflare credentials
	cloudflareEmail := getEnv("CLOUDFLARE_EMAIL")
	cloudflareAPIKey := getEnv("CLOUDFLARE_API_KEY")
	api, err := cloudflare.New(cloudflareAPIKey, cloudflareEmail)
	if err != nil {
		log.Fatalf("Failed to create Cloudflare API client: %v", err)
	}
	ctx := context.Background()

	// 3. Domains list
	domains := parseDomains()
	fmt.Printf("Processing domains: %s\n", strings.Join(domains, ", "))

	var hadError bool
	for _, domain := range domains {
		if err := updateDomain(ctx, api, currentIP, domain); err != nil {
			hadError = true
			log.Printf("Error updating %s: %v", domain, err)
			sendTelegramMessage(fmt.Sprintf("Error updating %s: %v", domain, err))
		}
	}
	if hadError {
		log.Fatal("One or more domains failed to update")
	}
}
