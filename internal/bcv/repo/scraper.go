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

	"Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"github.com/PuerkitoBio/goquery"
)

var (
	// fechaValorRegex matches "Fecha Valor: Martes, 19 Mayo 2026" and variants.
	fechaValorRegex = regexp.MustCompile(`(?i)fecha\s+valor[:\s]+(?:\w+,\s*)?(\d{1,2})\s+(\p{L}+)\s+(\d{4})`)

	// numericDateRegex is a fallback for DD/MM/YYYY or DD-MM-YYYY formats.
	numericDateRegex = regexp.MustCompile(`(\d{1,2})[/\-](\d{1,2})[/\-](\d{4})`)

	spanishMonths = map[string]time.Month{
		"enero": time.January, "febrero": time.February, "marzo": time.March,
		"abril": time.April, "mayo": time.May, "junio": time.June,
		"julio": time.July, "agosto": time.August, "septiembre": time.September,
		"octubre": time.October, "noviembre": time.November, "diciembre": time.December,
	}
)

type scraperRepo struct {
	client *http.Client
}

func NewScraper() ports.BcvScraper {
	return &scraperRepo{
		client: &http.Client{
			Timeout: 30 * time.Second,
			// BCV's certificate chain is incomplete; skip verification for this public read-only endpoint.
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
	}
}

func (s *scraperRepo) Scrape(ctx context.Context) (*models.ExchangeRate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.bcv.org.ve/", nil)
	if err != nil {
		return nil, fmt.Errorf("bcv scraper: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "es-VE,es;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bcv scraper: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bcv scraper: unexpected status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bcv scraper: parse HTML: %w", err)
	}

	rateStr := strings.TrimSpace(doc.Find("#dolar strong").First().Text())
	if rateStr == "" {
		return nil, fmt.Errorf("bcv scraper: USD rate not found (selector #dolar strong returned empty)")
	}

	rate, err := parseVenezuelanFloat(rateStr)
	if err != nil {
		return nil, fmt.Errorf("bcv scraper: parse rate %q: %w", rateStr, err)
	}

	return &models.ExchangeRate{
		Rate:      rate,
		Date:      extractFechaValor(doc),
		FetchedAt: time.Now().UTC(),
	}, nil
}

// parseVenezuelanFloat handles the Venezuelan number format (e.g. "91,6200" → 91.62).
func parseVenezuelanFloat(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// extractFechaValor parses the "Fecha Valor" field from the BCV page.
// It looks for the Spanish long format ("Martes, 19 Mayo 2026") that BCV uses
// for the value date field. Falls back to the most recent numeric date (DD/MM/YYYY).
func extractFechaValor(doc *goquery.Document) time.Time {
	bodyText := doc.Find("body").Text()

	if m := fechaValorRegex.FindStringSubmatch(bodyText); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, ok := spanishMonths[strings.ToLower(m[2])]
		year, _ := strconv.Atoi(m[3])
		if ok && day >= 1 && day <= 31 && year >= 2020 {
			return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		}
	}

	matches := numericDateRegex.FindAllStringSubmatch(bodyText, -1)
	var latest time.Time
	for _, m := range matches {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		if month < 1 || month > 12 || day < 1 || day > 31 || year < 2020 {
			continue
		}
		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if t.After(latest) {
			latest = t
		}
	}

	if latest.IsZero() {
		return time.Now().UTC().Truncate(24 * time.Hour)
	}
	return latest
}
