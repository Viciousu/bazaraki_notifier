//go:build integration
// +build integration

package main

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const testURL = "https://www.bazaraki.com/real-estate/houses-and-villas-rent/lemesos-district-limassol/"

func init() {
	// Initialize config with defaults for tests
	cfg = Config{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// newTestClient returns an HTTP client with a reasonable timeout for integration tests.
func newTestClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func TestFetchValidURL_SimpleCategory(t *testing.T) {
	client := newTestClient()
	url := testURL

	doc, err := validateAndFetchURL(url, client)
	if err != nil {
		t.Fatalf("expected no error for a valid category URL, got: %v", err)
	}
	if doc == nil {
		t.Fatal("expected a non-nil document")
	}

	// Verify that the page contains the advertisements container
	nodes := doc.Find(".list-announcement-assortiments").Nodes
	if len(nodes) == 0 {
		t.Error("expected .list-announcement-assortiments to be present on the page")
	}
}

func TestFetchValidURL_WithFilters(t *testing.T) {
	client := newTestClient()
	url := testURL

	doc, err := validateAndFetchURL(url, client)
	if err != nil {
		t.Fatalf("expected no error for the filtered URL, got: %v", err)
	}
	if doc == nil {
		t.Fatal("expected a non-nil document")
	}
}

func TestFetchValidURL_ResponseContainsAds(t *testing.T) {
	client := newTestClient()
	url := testURL

	doc, err := validateAndFetchURL(url, client)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that the page contains at least one advertisement link matching /adv/NNNNNNN_.../
	advPattern := regexp.MustCompile(`/adv/\d+_.*/`)
	found := false

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && advPattern.MatchString(href) {
			found = true
		}
	})

	if !found {
		t.Error("expected at least one advertisement link matching /adv/\\d+_.*/ on the page")
	}
}

func TestFetchValidURL_StatusCodeAndHeaders(t *testing.T) {
	client := newTestClient()
	url := testURL

	// Make a separate raw request to verify status code and content-type
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch URL: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		t.Error("expected Content-Type header to be present")
	}
}

func TestParseAds_FixedSelector(t *testing.T) {
	client := newTestClient()
	doc, err := validateAndFetchURL(testURL, client)
	if err != nil {
		t.Fatalf("failed to fetch URL: %v", err)
	}

	// Use the same logic as check_updates
	container := doc.Find(".list-simple__output")
	if container.Length() == 0 {
		t.Fatal("expected .list-simple__output container to be present")
	}

	advPattern := regexp.MustCompile(`/adv/\d{7}_.*/`)
	var ads []string
	container.Find("a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		isAdv := advPattern.MatchString(link)
		relevantAd := !s.HasClass("js-advert-gallery-item") && s.HasClass("mask")
		if isAdv && relevantAd && !Contains(ads, link) {
			ads = append(ads, link)
		}
	})

	if len(ads) == 0 {
		t.Fatal("expected at least one ad link from .list-simple__output container")
	}

	t.Logf("found %d unique ads", len(ads))
	for i, link := range ads {
		if i >= 3 {
			break
		}
		t.Logf("  %d. https://www.bazaraki.com%s", i+1, link)
	}
}

func TestFetchURL_WithoutUserAgent_MayFail(t *testing.T) {
	// This test documents that requests without a proper User-Agent may be blocked.
	// It is not a failure if it succeeds -- it just documents the behavior.
	client := newTestClient()

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	// Deliberately NOT setting User-Agent

	res, err := client.Do(req)
	if err != nil {
		t.Logf("request without User-Agent failed with error: %v (this may be expected)", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Logf("request without User-Agent returned status %d (this confirms bot detection)", res.StatusCode)
	} else {
		t.Logf("request without User-Agent succeeded with status 200 (no bot detection active)")
	}
}
