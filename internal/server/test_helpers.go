package server

import (
	"time"
)

// NewMockedSejmServer creates a SejmServer with pre-populated caches to avoid HTTP requests during testing.
// This ensures unit tests run fast and don't depend on external network connectivity.
func NewMockedSejmServer() *SejmServer {
	server := NewSejmServer()

	// Pre-populate all caches with mock data to prevent HTTP requests
	now := time.Now()
	expiry := now.Add(24 * time.Hour) // Long expiry to last through tests

	// Mock status types
	server.cache.mu.Lock()
	server.cache.StatusTypes = &CacheEntry{
		Data: []string{
			"obowiązujący",
			"uchylony",
			"nieobowiązujący",
			"wygaśnięcie aktu",
		},
		ExpiresAt: expiry,
	}

	// Mock document types
	server.cache.DocumentTypes = &CacheEntry{
		Data: []string{
			"Ustawa",
			"Konstytucja",
			"Rozporządzenie",
			"Dekret",
			"Uchwała",
		},
		ExpiresAt: expiry,
	}

	// Mock keywords
	server.cache.Keywords = &CacheEntry{
		Data: []string{
			"konstytucja",
			"sejm",
			"administracja",
			"podatki",
			"prawo",
			"sąd",
			"więcej",
			"łąw",
			"żółw",
		},
		ExpiresAt: expiry,
	}

	// Mock institutions
	server.cache.Institutions = &CacheEntry{
		Data: []string{
			"Sejm",
			"Senat",
			"Prezydent",
			"Trybunał Konstytucyjny",
			"Sąd Najwyższy",
			"Najwyższa Izba Kontroli",
			"Narodowy Bank Polski",
			"Wojewoda",
		},
		ExpiresAt: expiry,
	}

	// Mock popular acts
	server.cache.PopularActs = &CacheEntry{
		Data: []PopularAct{
			{
				Publisher:   "DU",
				Year:        "1997",
				Position:    "78",
				Title:       "Konstytucja Rzeczypospolitej Polskiej",
				Description: "Polish Constitution",
			},
			{
				Publisher:   "DU",
				Year:        "1964",
				Position:    "16",
				Title:       "Kodeks cywilny",
				Description: "Civil Code",
			},
		},
		ExpiresAt: expiry,
	}

	server.cache.mu.Unlock()

	return server
}
