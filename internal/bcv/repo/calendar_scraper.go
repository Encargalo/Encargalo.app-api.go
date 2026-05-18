package repo

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"github.com/PuerkitoBio/goquery"
)

// spanishDateRegex matches "19 de Mayo de 2026" or "19 de Mayo 2026".
var spanishDateRegex = regexp.MustCompile(`(?i)(\d{1,2})\s+de\s+(\p{L}+)\s+(?:de\s+)?(\d{4})`)

type calendarScraper struct {
	client *http.Client
}

func NewCalendarScraper() ports.BcvCalendarScraper {
	return &calendarScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
	}
}

func (s *calendarScraper) ScrapeHolidays(ctx context.Context) ([]time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://bcv.org.ve/bcv/calendario-bancario", nil)
	if err != nil {
		return nil, fmt.Errorf("bcv calendar: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "es-VE,es;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bcv calendar: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bcv calendar: unexpected status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bcv calendar: parse HTML: %w", err)
	}

	return parseHolidayDates(doc), nil
}

func parseHolidayDates(doc *goquery.Document) []time.Time {
	seen := make(map[string]bool)
	var dates []time.Time

	bodyText := doc.Find("body").Text()
	matches := spanishDateRegex.FindAllStringSubmatch(bodyText, -1)

	for _, m := range matches {
		day, _ := strconv.Atoi(m[1])
		month, ok := spanishMonths[strings.ToLower(m[2])]
		year, _ := strconv.Atoi(m[3])

		if !ok || day < 1 || day > 31 || year < 2020 {
			continue
		}

		t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		key := t.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		dates = append(dates, t)
	}

	return dates
}
