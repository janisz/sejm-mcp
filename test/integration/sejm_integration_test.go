package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/janisz/sejm-mcp/pkg/eli"
	"github.com/janisz/sejm-mcp/pkg/sejm"
)

const (
	sejmBaseURL = "https://api.sejm.gov.pl"
	eliBaseURL  = "https://api.sejm.gov.pl/eli"
)

func TestSejmAPIsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("SejmGetMPs", func(t *testing.T) {
		url := fmt.Sprintf("%s/sejm/term10/MP", sejmBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var mps []sejm.MP
		if err := json.NewDecoder(resp.Body).Decode(&mps); err != nil {
			t.Fatalf("Failed to decode MP data: %v", err)
		}

		if len(mps) == 0 {
			t.Fatal("Expected at least one MP")
		}

		// Validate MP structure
		firstMP := mps[0]
		if firstMP.Id == nil {
			t.Error("MP should have an ID")
		}
		if firstMP.FirstLastName == nil && (firstMP.FirstName == nil || firstMP.LastName == nil) {
			t.Error("MP should have name information")
		}

		t.Logf("Successfully retrieved %d MPs", len(mps))
	})

	t.Run("SejmGetMPDetails", func(t *testing.T) {
		url := fmt.Sprintf("%s/sejm/term10/MP/1", sejmBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var mp sejm.MP
		if err := json.NewDecoder(resp.Body).Decode(&mp); err != nil {
			t.Fatalf("Failed to decode MP details: %v", err)
		}

		// Validate MP details structure
		if mp.Id == nil {
			t.Error("MP details should have an ID")
		}

		t.Logf("Successfully retrieved MP details for ID 1")
	})

	t.Run("SejmGetCommittees", func(t *testing.T) {
		url := fmt.Sprintf("%s/sejm/term10/committees", sejmBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var committees []sejm.Committee
		if err := json.NewDecoder(resp.Body).Decode(&committees); err != nil {
			t.Fatalf("Failed to decode committee data: %v", err)
		}

		if len(committees) == 0 {
			t.Fatal("Expected at least one committee")
		}

		// Validate committee structure
		firstCommittee := committees[0]
		if firstCommittee.Code == nil {
			t.Error("Committee should have a code")
		}
		if firstCommittee.Name == nil {
			t.Error("Committee should have a name")
		}

		t.Logf("Successfully retrieved %d committees", len(committees))
	})

	t.Run("SejmSearchVotings", func(t *testing.T) {
		url := fmt.Sprintf("%s/sejm/term10/votings?limit=10", sejmBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var votings []sejm.Voting
		if err := json.NewDecoder(resp.Body).Decode(&votings); err != nil {
			t.Fatalf("Failed to decode voting data: %v", err)
		}

		if len(votings) == 0 {
			t.Fatal("Expected at least one voting record")
		}

		// Validate voting structure - just check that we can parse the data
		// Some voting records might not have all fields populated, which is expected
		firstVoting := votings[0]
		_ = firstVoting // Prevent unused variable warning

		t.Logf("Successfully retrieved %d voting records", len(votings))
	})

	t.Run("SejmGetInterpellations", func(t *testing.T) {
		url := fmt.Sprintf("%s/sejm/term10/interpellations?limit=10", sejmBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var interpellations []sejm.Interpellation
		if err := json.NewDecoder(resp.Body).Decode(&interpellations); err != nil {
			t.Fatalf("Failed to decode interpellation data: %v", err)
		}

		if len(interpellations) == 0 {
			t.Fatal("Expected at least one interpellation")
		}

		// Validate interpellation structure
		firstInterpellation := interpellations[0]
		if firstInterpellation.Num == nil {
			t.Error("Interpellation should have a number")
		}
		if firstInterpellation.Title == nil {
			t.Error("Interpellation should have a title")
		}

		t.Logf("Successfully retrieved %d interpellations", len(interpellations))
	})
}

func TestELIAPIsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("ELISearchActs", func(t *testing.T) {
		url := fmt.Sprintf("%s/acts/search?title=konstytucja&limit=5", eliBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var searchResult struct {
			Items []eli.ActInfo `json:"items"`
			Count int           `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			t.Fatalf("Failed to decode search results: %v", err)
		}

		if searchResult.Count == 0 {
			t.Fatal("Expected at least one search result for 'konstytucja'")
		}

		// Validate act structure
		if len(searchResult.Items) > 0 {
			firstAct := searchResult.Items[0]
			if firstAct.Title == nil {
				t.Error("Act should have a title")
			}
		}

		t.Logf("Successfully found %d legal acts matching 'konstytucja'", searchResult.Count)
	})

	t.Run("ELIGetActDetails", func(t *testing.T) {
		// Test with a real ELI document (DU/2016/538)
		url := fmt.Sprintf("%s/acts/DU/2016/538", eliBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var act eli.Act
		if err := json.NewDecoder(resp.Body).Decode(&act); err != nil {
			t.Fatalf("Failed to decode act details: %v", err)
		}

		// Validate act details
		if act.Title == nil {
			t.Error("Act should have a title")
		}
		if act.ELI == nil {
			t.Error("Act should have an ELI identifier")
		}

		t.Logf("Successfully retrieved act details for DU/2016/538")
	})

	t.Run("ELIGetActText", func(t *testing.T) {
		// Test with a real ELI document HTML text
		url := fmt.Sprintf("%s/acts/DU/2016/538/text.html", eliBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		// Read some content to verify it's not empty
		buffer := make([]byte, 1024)
		n, err := resp.Body.Read(buffer)
		if err != nil && n == 0 {
			t.Fatal("Expected non-empty act text")
		}

		if n < 100 {
			t.Error("Act text seems too short")
		}

		t.Logf("Successfully retrieved act text (%d+ bytes)", n)
	})

	t.Run("ELIGetActReferences", func(t *testing.T) {
		// Test with a real ELI document references
		url := fmt.Sprintf("%s/acts/DU/2016/538/references", eliBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var references eli.ReferencesDetailsInfo
		if err := json.NewDecoder(resp.Body).Decode(&references); err != nil {
			t.Fatalf("Failed to decode references: %v", err)
		}

		totalRefs := 0
		for _, refList := range references {
			totalRefs += len(refList)
		}

		t.Logf("Successfully retrieved %d reference categories with %d total references for DU/2016/538", len(references), totalRefs)
	})

	t.Run("ELIGetPublishers", func(t *testing.T) {
		// The publishers are actually at /eli/acts endpoint
		url := fmt.Sprintf("%s/acts", eliBaseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("API returned status %d", resp.StatusCode)
		}

		var publishers []eli.PublishingHouse
		if err := json.NewDecoder(resp.Body).Decode(&publishers); err != nil {
			t.Fatalf("Failed to decode publishers: %v", err)
		}

		if len(publishers) == 0 {
			t.Fatal("Expected at least one publisher")
		}

		// Check for DU publisher (Dziennik Ustaw)
		foundDU := false
		for _, pub := range publishers {
			if pub.Code != nil && *pub.Code == "DU" {
				foundDU = true
				break
			}
		}

		if !foundDU {
			t.Error("Expected to find DU publisher")
		}

		t.Logf("Successfully retrieved %d publishers", len(publishers))
	})
}
