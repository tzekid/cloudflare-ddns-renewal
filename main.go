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

func main() {
	// 1. Get your current public IP.
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

	// 2. Fetch Cloudflare credentials from environment variables.
	cloudflareEmail := getEnv("CLOUDFLARE_EMAIL")
	cloudflareAPIKey := getEnv("CLOUDFLARE_API_KEY")
	// Optionally, if you want to use an API token instead, set CLOUDFLARE_API_TOKEN and adjust accordingly.
	// cloudflareAPIToken := getEnv("CLOUDFLARE_API_TOKEN")

	// Create a Cloudflare API client using API key and email.
	api, err := cloudflare.New(cloudflareAPIKey, cloudflareEmail)
	if err != nil {
		log.Fatalf("Failed to create Cloudflare API client: %v", err)
	}
	// Alternatively, to use an API token:
	// api, err := cloudflare.NewWithAPIToken(cloudflareAPIToken)
	// if err != nil {
	//     log.Fatalf("Failed to create Cloudflare API client with token: %v", err)
	// }

	ctx := context.Background()

	// 3. List zones.
	zones, err := api.ListZones(ctx)
	if err != nil {
		log.Fatalf("Error listing zones: %v", err)
	}
	var zoneID string
	for _, zone := range zones {
		// Adjust the selection logic if you have multiple zones.
		zoneID = zone.ID // This example simply uses the last zone found.
	}
	if zoneID == "" {
		log.Fatalf("No zones found")
	}

	// Create a resource container for the zone.
	rc := &cloudflare.ResourceContainer{
		Identifier: zoneID,
		Type:       "zone",
	}

	// 4. Define your domain and its wildcard.
	// These can also be parameterized or set via environment variables if needed.
	domain := "plosca.ru"
	wildcard := "*." + domain

	// 5. Retrieve DNS records for the zone.
	params := cloudflare.ListDNSRecordsParams{} // empty search parameters
	recs, _, err := api.ListDNSRecords(ctx, rc, params)
	if err != nil {
		log.Fatalf("Error fetching DNS records: %v", err)
	}
	var domainRec *cloudflare.DNSRecord
	var wildcardRec *cloudflare.DNSRecord
	for _, rec := range recs {
		if rec.Name == domain && rec.Type == "A" {
			domainRec = &rec
		} else if rec.Name == wildcard && rec.Type == "A" {
			wildcardRec = &rec
		}
	}
	if domainRec == nil {
		log.Fatalf("A record for %s not found", domain)
	}
	if wildcardRec == nil {
		log.Fatalf("A record for %s not found", wildcard)
	}

	// 6. Update the domain record if needed.
	if domainRec.Content == currentIP {
		fmt.Printf("%s is already up to date\n", domain)
		// Uncomment to notify even when no changes occur:
		// sendTelegramMessage("No IP changes for " + domain)
	} else {
		fmt.Printf("Updating %s from %s to %s\n", domain, domainRec.Content, currentIP)
		updateParams := cloudflare.UpdateDNSRecordParams{
			ID:      domainRec.ID,
			Type:    "A",
			Name:    domain,
			Content: currentIP,
			TTL:     domainRec.TTL,
			Proxied: domainRec.Proxied,
		}
		_, err = api.UpdateDNSRecord(ctx, rc, updateParams)
		if err != nil {
			log.Fatalf("Failed to update DNS record for %s: %v", domain, err)
		}
		sendTelegramMessage(fmt.Sprintf("IP for %s updated to %s", domain, currentIP))
	}

	// 7. Update the wildcard record if needed.
	if wildcardRec.Content == currentIP {
		fmt.Printf("%s is already up to date\n", wildcard)
		// Uncomment to notify even when no changes occur:
		// sendTelegramMessage(fmt.Sprintf("%s is already up to date", wildcard))
	} else {
		fmt.Printf("Updating %s from %s to %s\n", wildcard, wildcardRec.Content, currentIP)
		updateParams := cloudflare.UpdateDNSRecordParams{
			ID:      wildcardRec.ID,
			Type:    "A",
			Name:    wildcard,
			Content: currentIP,
			TTL:     wildcardRec.TTL,
			Proxied: wildcardRec.Proxied,
		}
		_, err = api.UpdateDNSRecord(ctx, rc, updateParams)
		if err != nil {
			log.Fatalf("Failed to update DNS record for %s: %v", wildcard, err)
		}
		sendTelegramMessage(fmt.Sprintf("IP for %s updated to %s", wildcard, currentIP))
	}
}
