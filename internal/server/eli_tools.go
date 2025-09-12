// Package server implements ELI (Polish Legal Information System) API handlers for the sejm-mcp server.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
	"github.com/janisz/sejm-mcp/pkg/eli"
	"github.com/mark3labs/mcp-go/mcp"
)

const eliBaseURL = "https://api.sejm.gov.pl/eli"

var eliLegalStatuses = []string{
	"akt indywidualny", "akt jednorazowy", "akt objęty tekstem jednolitym",
	"akt posiada tekst jednolity", "bez statusu", "brak mocy prawnej",
	"nieobowiązujący - przyczyna nieustalona", "nieobowiązujący - uchylona podstawa prawna",
	"obowiązujący", "tekst jednolity dla aktu jednorazowego", "uchylony",
	"uchylony wykazem", "uznany za uchylony", "wydane z naruszeniem prawa", "wygaśnięcie aktu",
}

var eliDocumentTypes = []string{
	"Oświadczenie", "Umowa zbiorowa", "Lista", "Konwencja", "Komunikat", "Układ",
	"Orędzie", "Zalecenie", "Dokument wypowiedzenia", "Umowa", "Wykaz",
	"Oświadczenie rządowe", "Statut", "Ustawa", "Raport", "Apel", "Sprostowanie",
	"Pismo okólne", "Okólnik", "Porozumienie", "Obwieszczenie", "Reskrypt",
	"Przepisy", "Dekret", "Traktat", "Rozkaz", "Instrukcja", "Sprawozdanie",
	"Opinia", "Umowa międzynarodowa", "Wyjaśnienie", "Wytyczne", "Decyzja",
	"Wypis", "Stanowisko", "Przepisy wykonawcze", "Rezolucja", "Rozporządzenie",
	"Karta", "Zawiadomienie", "Akt", "Uchwała", "Orzeczenie", "Ogłoszenie",
	"Deklaracja", "Regulamin", "Protokół", "Zarządzenie", "Informacja",
	"Postanowienie", "Interpretacja",
}

// StandardResponse provides a consistent format for all API responses
type StandardResponse struct {
	Operation   string
	Status      string
	Summary     []string
	Data        []string
	NextActions []string
	Note        string
}

// Format returns a standardized response string
func (sr StandardResponse) Format() string {
	var result strings.Builder

	// Header with operation and status
	result.WriteString(fmt.Sprintf("%s - %s", sr.Operation, sr.Status))

	// Summary section
	if len(sr.Summary) > 0 {
		result.WriteString("\n\nSummary:")
		for _, item := range sr.Summary {
			result.WriteString(fmt.Sprintf("\n• %s", item))
		}
	}

	// Data section
	if len(sr.Data) > 0 {
		result.WriteString("\n\nResults:")
		for _, item := range sr.Data {
			result.WriteString(fmt.Sprintf("\n%s", item))
		}
	}

	// Next actions section
	if len(sr.NextActions) > 0 {
		result.WriteString("\n\nNext Actions:")
		for _, action := range sr.NextActions {
			result.WriteString(fmt.Sprintf("\n• %s", action))
		}
	}

	// Note section
	if sr.Note != "" {
		result.WriteString(fmt.Sprintf("\n\nNote: %s", sr.Note))
	}

	return result.String()
}

// buildCrossReferenceHints analyzes search results and suggests related navigation
func buildCrossReferenceHints(acts []eli.Act, baseActions []string) []string {
	actions := make([]string, 0, len(baseActions)+10) // Reserve capacity for base + additional actions
	actions = append(actions, baseActions...)

	// Analyze the types of acts found and suggest related searches
	hasConstitution := false
	hasCodes := false
	hasRegulations := false

	for _, act := range acts {
		if act.Title != nil {
			title := strings.ToLower(*act.Title)
			if strings.Contains(title, "konstytucja") {
				hasConstitution = true
			}
			if strings.Contains(title, "kodeks") {
				hasCodes = true
			}
		}
		if act.Type != nil && strings.Contains(strings.ToLower(*act.Type), "rozporządzenie") {
			hasRegulations = true
		}
	}

	// Add specific cross-reference suggestions based on content
	if hasConstitution {
		actions = append(actions, "Explore Constitutional law relationships: eli_get_act_references for Constitution (DU/1997/78)")
		actions = append(actions, "Find Constitutional amendments: eli_search_acts with type='ustawa' and title='konstytucja'")
	}

	if hasCodes {
		actions = append(actions, "Explore related legal codes: eli_search_acts with title='kodeks'")
		actions = append(actions, "Find implementing regulations for codes: eli_search_acts with type='rozporządzenie'")
	}

	if hasRegulations {
		actions = append(actions, "Find parent laws for regulations: eli_get_act_references to trace legal basis")
		actions = append(actions, "Search for related regulations: eli_search_acts with same publisher and type='rozporządzenie'")
	}

	// Add reference type guidance based on hardcoded reference types
	actions = append(actions, "")
	actions = append(actions, "Available legal relationship types in eli_get_act_references:")
	actions = append(actions, "• 'Akty zmieniające' - Acts that amend this law")
	actions = append(actions, "• 'Akty uchylające' - Acts that repeal this law")
	actions = append(actions, "• 'Akty wykonawcze' - Implementing regulations")
	actions = append(actions, "• 'Podstawa prawna' - Legal basis and foundation")
	actions = append(actions, "• 'Tekst jednolity dla aktu' - Consolidated text information")
	actions = append(actions, "• 'Orzeczenie TK' - Constitutional Court rulings")

	return actions
}

// parseOffsetLimit parses offset and limit strings into integers with defaults
func parseOffsetLimit(offset, limit string) (int, int) {
	offsetInt := 0
	if offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			offsetInt = parsed
		}
	}

	limitInt := 20
	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			limitInt = parsed
		}
	}
	return offsetInt, limitInt
}

// buildPaginationHints creates pagination guidance based on current offset, limit and total results
func buildPaginationHints(offset, limit string, totalCount int) []string {
	var hints []string
	offsetInt, limitInt := parseOffsetLimit(offset, limit)

	// Calculate current page info
	currentEnd := offsetInt + limitInt
	if currentEnd > totalCount {
		currentEnd = totalCount
	}

	// Add navigation hints
	if offsetInt > 0 {
		prevOffset := offsetInt - limitInt
		if prevOffset < 0 {
			prevOffset = 0
		}
		hints = append(hints, fmt.Sprintf("Previous page: use offset='%d' (results %d-%d)",
			prevOffset, prevOffset+1, min2(prevOffset+limitInt, totalCount)))
	}

	if currentEnd < totalCount {
		nextOffset := offsetInt + limitInt
		nextEnd := nextOffset + limitInt
		if nextEnd > totalCount {
			nextEnd = totalCount
		}
		hints = append(hints, fmt.Sprintf("Next page: use offset='%d' (results %d-%d)",
			nextOffset, nextOffset+1, nextEnd))
	}

	// Add sorting hints
	if len(hints) > 0 {
		hints = append(hints, "Sort options: sort_by='date' (newest first), sort_by='title' (A-Z), sort_by='year,desc' (recent years first)")
	}

	return hints
}

func (s *SejmServer) registerELITools() {
	s.server.AddTool(mcp.Tool{
		Name:        "eli_search_acts",
		Description: "Search Poland's comprehensive legal acts database using European Legislation Identifier (ELI) standards. This powerful tool searches through all published Polish legal documents including laws, regulations, decrees, ordinances, and constitutional acts. Returns standardized metadata with ELI identifiers, titles, publication details, legal status, and document types. Essential for legal research, citation verification, regulatory compliance analysis, academic legal studies, and building legal knowledge bases. Use this as the primary entry point for legal document discovery - it's like Google but specifically for Polish legal documents.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Search keywords in document titles. Use Polish terms for best results (e.g., 'konstytucja' for constitution, 'kodeks' for code, 'ustawa' for law, 'rozporządzenie' for regulation). Supports partial matches and multiple keywords. Examples: 'kodeks pracy' (labor code), 'prawo autorskie' (copyright law), 'ochrona danych' (data protection).",
				},
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Official publisher code. Key publishers: 'DU' (Dziennik Ustaw - main Journal of Laws for primary legislation), 'MP' (Monitor Polski - for secondary legislation and administrative acts), ministry codes like 'ME' (Ministry of Economy). Use 'DU' for major laws and constitutional acts. To discover all available publisher codes, use eli_get_publishers tool.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year (e.g., '2020', '1997', '2018'). Use specific years to find legislation from particular periods. Key years: 1997 (current Constitution), 1964 (Civil Code), 1974 (Labor Code), 2018 (GDPR implementation).",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Legal document type. Common types: 'ustawa' (statute/law), 'konstytucja' (constitution), 'rozporządzenie' (regulation), 'dekret' (decree), 'zarządzenie' (directive). Use specific types to narrow search to particular kinds of legal instruments. To discover all available document types, use eli_get_types tool.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum search results to return (default: 50). Use '10-20' for quick searches, '50-100' for comprehensive research, '200+' for extensive legal analysis. Larger limits provide more complete coverage but take longer to process.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Number of results to skip for pagination (default: 0). Use with limit for efficient browsing through large result sets. Example: offset='50' with limit='25' gets results 51-75.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort results by field: 'date' (publication date), 'title' (alphabetical), 'year' (publication year), 'publisher' (publisher code). Default is relevance-based ordering. Combine with sort_dir to control order.",
				},
				"sort_dir": map[string]interface{}{
					"type":        "string",
					"description": "Sort direction: 'asc' (ascending) or 'desc' (descending). Default is 'desc' for dates (newest first) and 'asc' for text fields (A-Z). Only used when sort_by is specified.",
				},
				"date_from": map[string]interface{}{
					"type":        "string",
					"description": "Start date for announcement date search in YYYY-MM-DD format (e.g., '2020-01-01'). Only returns acts announced from this date onwards. Use with date_to for date range searches.",
				},
				"date_to": map[string]interface{}{
					"type":        "string",
					"description": "End date for announcement date search in YYYY-MM-DD format (e.g., '2023-12-31'). Only returns acts announced up to this date. Use with date_from for date range searches.",
				},
				"in_force": map[string]interface{}{
					"type":        "string",
					"description": "Filter by legal status: '1' to search only for acts currently in force, empty/omit to search all acts regardless of status. Useful for finding only active legislation.",
				},
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "Search for specific legal keywords/concepts in act content, separated by commas. Different from title search - searches deeper content and official legal keywords. Examples: 'ochrona przyrody' (nature protection), 'kodeks wyborczy' (electoral code), 'administracja samorządowa' (local government administration), 'prawo pracy' (labor law), 'podatek dochodowy' (income tax), 'ochrona danych' (data protection), 'bezpieczeństwo publiczne' (public safety). To discover all available keywords, use eli_get_keywords tool. Keywords are official legal concept tags assigned to acts.",
				},
			},
		},
	}, s.handleSearchActs)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_act_details",
		Description: "Retrieve comprehensive metadata and legal information about a specific Polish legal act using its official publication identifiers. Returns detailed legal document profile including official title, ELI identifier, publication and effective dates, current legal status (in force/repealed/expired), document type classification, issuing institution, legal keywords, amendment history, available text formats, and related document counts. Essential for legal citation verification, regulatory compliance checking, legal research validation, academic legal studies, and building authoritative legal databases. Use this when you have specific legal act coordinates from search results or citations.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Official publisher code from Polish legal system. Primary publishers: 'DU' (Dziennik Ustaw - Journal of Laws for major legislation including Constitution, codes, primary laws), 'MP' (Monitor Polski - for secondary legislation, ministerial orders), 'DzUrz' (ministry-specific gazettes). Get this from eli_search_acts results or legal citations. Required for precise document identification.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year as 4-digit string (e.g., '1997', '2020', '2018'). This is the year the document was officially published, not necessarily when it became effective. Critical for legal citation accuracy. Examples: Polish Constitution is '1997', current Civil Code dates to '1964', GDPR implementation is '2018'.",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Sequential position number within the publication year and publisher. Each legal document gets a unique position number when published (e.g., '78' for Constitution in DU 1997, '16' for Civil Code in DU 1964). This ensures precise document identification within the legal system. Get from search results or legal citations.",
				},
				"detailed": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Set to 'true' to get complete metadata JSON. Default is summary view to reduce token usage. Use detailed view only when you need full legal metadata for analysis.",
				},
			},
			Required: []string{"publisher", "year", "position"},
		},
	}, s.handleGetActDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_act_text",
		Description: "Download the complete official text of a Polish legal act in PDF or plain text format. PDF format delivers the official publication-quality document suitable for citations and archival. TEXT format extracts plain text from PDF, providing clean text perfect for AI processing. HTML format is rarely available in the Polish ELI system - most documents are only published in PDF format. The text includes the full legal content as published, with proper legal structure, amendment annotations, and official formatting. Critical for legal analysis, AI-powered legal research, compliance checking, academic studies, and legal document processing.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Official publisher code. Use 'DU' for major laws (Constitution, codes, primary legislation), 'MP' for secondary legislation and administrative acts, or specific ministry codes. Must match the publisher from the act's official citation or eli_get_act_details results.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year as 4-digit string. Must exactly match the official publication year from legal citations or search results. Examples: '1997' for Constitution, '1964' for Civil Code, '2018' for GDPR implementation law.",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Exact position number from the official publication. This unique identifier ensures you get the precise legal document version. Obtain from eli_search_acts results or legal citations (e.g., '78' for Constitution, '16' for Civil Code).",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Document format: 'pdf' (recommended) for official publication-quality document, 'text' for plain text extracted from PDF ideal for AI processing, or 'html' for structured text (rarely available - most Polish legal documents are only published in PDF format).",
				},
				"page": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Page number to retrieve (1-based, for text/html formats only). Use this to get specific pages and avoid large responses. Example: '1' for first page, '5' for fifth page. If not specified, returns full document.",
				},
				"pages_per_chunk": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Number of pages per chunk (default: 5, max: 20, for text/html formats only). Use with 'page' parameter to control response size. Example: '3' to get 3 pages starting from the specified page. Useful for reading long documents in manageable chunks.",
				},
				"show_page_info": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Set to 'true' to show page count and navigation info without retrieving full text (for text/html formats). Useful for understanding document structure before reading specific pages.",
				},
			},
			Required: []string{"publisher", "year", "position"},
		},
	}, s.handleGetActText)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_act_references",
		Description: "Explore the complex legal relationship network between Polish legal acts through citations, amendments, repeals, and references. Returns comprehensive mapping of how legal documents connect to each other including: outgoing references (laws this act cites), incoming citations (laws that cite this act), amendments and modifications, implementing regulations, repealing acts, and legal basis relationships. Each reference includes relationship type, direction, target document details, specific provisions cited, and dates. Essential for legal dependency analysis, understanding legislative history, tracking law evolution, regulatory impact assessment, legal research, and building comprehensive legal knowledge graphs.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Publisher code of the source legal act to analyze. Use 'DU' for major legislation, 'MP' for administrative acts. The reference analysis will show how this specific document connects to the broader legal system through citations and amendments.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year of the source act. Legal references are tracked from the original publication date forward, showing how the law has been cited, amended, or affected by subsequent legislation over time.",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Position number of the source act. This identifies the exact legal document whose legal relationships you want to explore. Major acts often have extensive reference networks showing their importance in the legal system.",
				},
			},
			Required: []string{"publisher", "year", "position"},
		},
	}, s.handleGetActReferences)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_publishers",
		Description: "Retrieve comprehensive directory of all official Polish legal document publishers in the ELI system. Returns detailed information about each publishing authority including publisher codes, official names (Polish and English), descriptions, publication scope, document counts, active date ranges, and website links. Publishers represent different levels and types of legal authority: national legislature (DU), government administration (MP), individual ministries (ministry-specific codes), regional authorities, and specialized agencies. Essential for understanding the Polish legal publication system, determining appropriate search parameters, validating legal citations, building comprehensive legal databases, and navigating the hierarchical structure of Polish legal documentation. Use this as reference when working with other ELI tools.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
		},
	}, s.handleGetPublishers)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_search_act_content",
		Description: "Search for specific text within a Polish legal act and get precise page locations. This powerful tool downloads the complete legal document, searches for your specified terms, and returns a detailed map showing exactly which pages contain each search term. Perfect for quickly locating specific provisions, articles, concepts, or keywords within large legal documents without reading the entire text. Essential for legal research, finding relevant sections, preparing citations, analyzing specific legal concepts, and navigating complex legislation efficiently. Much faster than manual searching through hundreds of pages.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Official publisher code (e.g., 'DU' for major laws, 'MP' for regulations). Must match exactly with the legal act's publication details.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year as 4-digit string (e.g., '1997', '2020'). Must match the official publication year of the legal act.",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Exact position number from official publication (e.g., '78' for Constitution, '483' for specific laws). Must match the official position identifier.",
				},
				"search_terms": map[string]interface{}{
					"type":        "string",
					"description": "Search terms separated by commas. Can include single words, phrases, article numbers, or legal concepts. Examples: 'konstytucja,artykuł 15,prawa człowieka' or 'podatek,VAT,zwolnienie'. Case-insensitive search with Polish character support.",
				},
				"context_chars": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Number of characters to show around each match for context (default: 100, max: 500). Higher values provide more context but use more tokens.",
				},
				"max_matches_per_term": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Maximum number of matches to show per search term (default: 10, max: 50). Helps limit response size for common terms.",
				},
			},
			Required: []string{"publisher", "year", "position", "search_terms"},
		},
	}, s.handleSearchActContent)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_keywords",
		Description: "Retrieve comprehensive list of all available legal keywords used in the Polish ELI acts database. Returns a complete directory of official legal concept tags that can be used for keyword searches. These keywords represent standardized legal terminology and subject classifications used to categorize Polish legal acts. Essential for discovering searchable legal concepts, building comprehensive legal searches, understanding legal topic coverage, and ensuring accurate keyword-based searches. Use this to find the exact keyword terms for eli_search_acts keyword parameter. Keywords are cached for performance and updated periodically.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sort": map[string]interface{}{
					"type":        "string",
					"description": "Sort keywords alphabetically: 'asc' for A-Z (default), 'desc' for Z-A. Helps organize keywords for browsing.",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter keywords containing specific text (e.g., 'prawo' to find all law-related keywords, 'podatek' for tax-related). Case-insensitive partial matching.",
				},
			},
		},
	}, s.handleGetKeywords)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_types",
		Description: "Retrieve comprehensive list of all available legal document types in the Polish ELI system. Returns standardized document type classifications used to categorize Polish legal acts such as 'Ustawa' (statute), 'Rozporządzenie' (regulation), 'Dekret' (decree), 'Uchwała' (resolution), etc. Essential for discovering valid document types for eli_search_acts type parameter, understanding the Polish legal document hierarchy, building comprehensive searches, and ensuring accurate type-based filtering. Use this reference when working with document type searches.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sort": map[string]interface{}{
					"type":        "string",
					"description": "Sort document types alphabetically: 'asc' for A-Z (default), 'desc' for Z-A. Helps organize types for browsing.",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter types containing specific text (e.g., 'ustawa' for laws, 'rozporządzenie' for regulations). Case-insensitive partial matching.",
				},
			},
		},
	}, s.handleGetTypes)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_statuses",
		Description: "Retrieve comprehensive list of all available legal status classifications in the Polish ELI system. Returns standardized legal status categories such as 'obowiązujący' (in force), 'uchylony' (repealed), 'nieobowiązujący' (not in force), 'wygaśnięcie aktu' (expired), etc. Essential for discovering valid legal statuses, understanding document lifecycle states, building status-based searches, and filtering acts by their current legal validity. Use this reference when working with legal status searches and compliance checking.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sort": map[string]interface{}{
					"type":        "string",
					"description": "Sort legal statuses alphabetically: 'asc' for A-Z (default), 'desc' for Z-A. Helps organize statuses for browsing.",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter statuses containing specific text (e.g., 'obowiązujący' for active laws, 'uchylony' for repealed). Case-insensitive partial matching.",
				},
			},
		},
	}, s.handleGetStatuses)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_list_acts",
		Description: "Retrieve basic listing of legal acts from the Polish ELI database with pagination support. Returns essential metadata for acts including titles, publishers, years, and identifiers. Use this for browsing available acts, getting overview of legal documents, or as starting point for more detailed searches. Complements eli_search_acts by providing simple listing functionality without search criteria requirements.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of acts to return (default: 50, max: 500). Use higher values for comprehensive listings.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Number of results to skip for pagination (default: 0). Use with limit for browsing through large collections.",
				},
			},
		},
	}, s.handleListActs)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_acts_by_publisher",
		Description: "Retrieve all legal acts published by a specific publisher authority. Returns comprehensive listing of acts from publishers like 'DU' (Dziennik Ustaw), 'MP' (Monitor Polski), or ministry codes. Essential for analyzing publisher-specific legislation, understanding institutional legal output, researching ministry-specific regulations, and building publisher-focused legal databases. Use eli_get_publishers to discover available publisher codes.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Publisher code (e.g., 'DU', 'MP', ministry codes). Get codes from eli_get_publishers. Required parameter.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of acts to return (default: 100). Use higher values for comprehensive publisher analysis.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Number of results to skip for pagination (default: 0). Use with limit for browsing large publisher collections.",
				},
			},
			Required: []string{"publisher"},
		},
	}, s.handleGetActsByPublisher)

	s.server.AddTool(mcp.Tool{
		Name:        "eli_get_acts_by_year",
		Description: "Retrieve all legal acts published by a specific publisher in a given year. Returns comprehensive yearly legislation from specified publisher authorities. Essential for temporal legal analysis, understanding yearly legislative output, researching historical legal development, tracking regulatory activity by year, and building time-series legal databases. Useful for legislative trend analysis and historical legal research.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"publisher": map[string]interface{}{
					"type":        "string",
					"description": "Publisher code (e.g., 'DU', 'MP'). Get codes from eli_get_publishers. Required parameter.",
				},
				"year": map[string]interface{}{
					"type":        "string",
					"description": "Publication year (e.g., '2020', '1997'). Required parameter.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of acts to return (default: 100). Use higher values for comprehensive yearly analysis.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Number of results to skip for pagination (default: 0). Use with limit for browsing large yearly collections.",
				},
			},
			Required: []string{"publisher", "year"},
		},
	}, s.handleGetActsByYear)
}

func (s *SejmServer) handleSearchActs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := make(map[string]string)

	title := request.GetString("title", "")
	if title != "" {
		params["title"] = title
	}

	publisher := request.GetString("publisher", "")
	if publisher != "" {
		params["publisher"] = publisher
	}

	year := request.GetString("year", "")
	if year != "" {
		params["year"] = year
	}

	docType := request.GetString("type", "")
	if docType != "" {
		params["type"] = docType
	}

	limit := request.GetString("limit", "20") // Reduced default to avoid context overflow
	params["limit"] = limit

	offset := request.GetString("offset", "")
	if offset != "" {
		params["offset"] = offset
	}

	sortBy := request.GetString("sort_by", "")
	if sortBy != "" {
		params["sort"] = sortBy

		// Handle sort direction
		sortDir := request.GetString("sort_dir", "")
		if sortDir != "" {
			// Validate sort direction
			if sortDir == "asc" || sortDir == "desc" {
				// For ELI API, append direction to sort field with comma
				params["sort"] = sortBy + "," + sortDir
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid sort_dir '%s'. Must be 'asc' or 'desc'.", sortDir)), nil
			}
		}
	}

	// Add new search parameters
	dateFrom := request.GetString("date_from", "")
	if dateFrom != "" {
		params["dateFrom"] = dateFrom
	}

	dateTo := request.GetString("date_to", "")
	if dateTo != "" {
		params["dateTo"] = dateTo
	}

	inForce := request.GetString("in_force", "")
	if inForce != "" {
		params["inForce"] = inForce
	}

	keyword := request.GetString("keyword", "")
	if keyword != "" {
		params["keyword"] = keyword
	}

	s.logger.Info("eli_search_acts called",
		slog.String("title", title),
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("type", docType),
		slog.String("limit", limit),
		slog.String("offset", offset),
		slog.String("sort_by", sortBy),
		slog.String("sort_dir", request.GetString("sort_dir", "")),
		slog.String("date_from", dateFrom),
		slog.String("date_to", dateTo),
		slog.String("in_force", inForce),
		slog.String("keyword", keyword))

	// Validate that at least one search parameter is provided
	// Count only actual search parameters (not pagination/sorting parameters)
	searchParamCount := 0
	if title != "" {
		searchParamCount++
	}
	if publisher != "" {
		searchParamCount++
	}
	if year != "" {
		searchParamCount++
	}
	if docType != "" {
		searchParamCount++
	}
	if keyword != "" {
		searchParamCount++
	}
	if dateFrom != "" || dateTo != "" {
		searchParamCount++
	}
	if inForce != "" {
		searchParamCount++
	}

	if searchParamCount == 0 {
		return mcp.NewToolResultError("Please provide at least one search parameter (title, publisher, year, type, keyword, date range, or in_force status) to search legal acts. Examples: 'konstytucja' for title, 'DU' for publisher, 'ochrona danych' for keyword, or '1' for in_force to find only active laws."), nil
	}

	// Validate publisher code if provided
	if publisher != "" {
		isValid, suggestions, err := s.validatePublisher(ctx, publisher)
		if err != nil {
			s.logger.Warn("Publisher validation failed", slog.String("publisher", publisher), slog.Any("error", err))
			// Log error but don't fail the search - continue with provided publisher
		} else if !isValid {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid publisher code '%s'. %s", publisher, strings.Join(suggestions, "\n"))), nil
		}
	}

	// Validate document type if provided
	if docType != "" {
		isValid, suggestions, err := s.validateDocumentType(docType)
		if err != nil {
			s.logger.Warn("Document type validation failed", slog.String("docType", docType), slog.Any("error", err))
			// Log error but don't fail the search - continue with provided type
		} else if !isValid {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid document type '%s'. %s", docType, strings.Join(suggestions, "\n"))), nil
		}
	}

	endpoint := fmt.Sprintf("%s/acts/search", eliBaseURL)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search Polish legal acts database: %v. Please verify your search parameters are valid.", err)), nil
	}

	var searchResult struct {
		Items []eli.Act `json:"items"`
		Count int       `json:"count"`
	}
	if err := json.Unmarshal(data, &searchResult); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse legal acts search results: %v. The ELI API may have returned unexpected data format.", err)), nil
	}

	// Build search criteria summary
	var criteria []string
	if title != "" {
		criteria = append(criteria, fmt.Sprintf("Title keywords: '%s'", title))
	}
	if publisher != "" {
		criteria = append(criteria, fmt.Sprintf("Publisher: %s", publisher))
	}
	if year != "" {
		criteria = append(criteria, fmt.Sprintf("Year: %s", year))
	}
	if docType != "" {
		criteria = append(criteria, fmt.Sprintf("Document type: %s", docType))
	}

	// Add pagination and sorting info
	if offset != "" {
		criteria = append(criteria, fmt.Sprintf("Offset: %s results skipped", offset))
	}
	if sortBy != "" {
		sortInfo := fmt.Sprintf("Sorted by: %s", sortBy)
		if sortDir := request.GetString("sort_dir", ""); sortDir != "" {
			sortInfo += fmt.Sprintf(" (%s)", sortDir)
		}
		criteria = append(criteria, sortInfo)
	}

	criteria = append(criteria, fmt.Sprintf("Found %d legal acts", searchResult.Count))

	if searchResult.Count == 0 {
		// Provide intelligent suggestions based on search terms
		var suggestions []string

		// Analyze search terms to provide targeted suggestions
		if title != "" {
			titleLower := strings.ToLower(title)
			if strings.Contains(titleLower, "konstytucja") {
				suggestions = append(suggestions, "eli_search_acts with title='konstytucja' and publisher='DU' and year='1997' (Polish Constitution)")
			} else if strings.Contains(titleLower, "kodeks") {
				suggestions = append(suggestions, "eli_search_acts with title='kodeks cywilny' (Civil Code)")
				suggestions = append(suggestions, "eli_search_acts with title='kodeks pracy' (Labor Code)")
				suggestions = append(suggestions, "eli_search_acts with title='kodeks karny' (Criminal Code)")
			} else if strings.Contains(titleLower, "gdpr") || strings.Contains(titleLower, "ochrona danych") {
				suggestions = append(suggestions, "eli_search_acts with title='ochrona danych' and year='2018' (GDPR implementation)")
			} else {
				suggestions = append(suggestions, fmt.Sprintf("Try broader search: remove specific words from '%s'", title))
				suggestions = append(suggestions, "Use root words: 'prawo' instead of 'prawny', 'ustawa' instead of specific law names")
			}
		}

		// Publisher-specific suggestions
		if publisher == "" {
			suggestions = append(suggestions, "Try publisher='DU' for major laws (Constitution, codes, primary legislation)")
			suggestions = append(suggestions, "Try publisher='MP' for secondary legislation and administrative acts")
		}

		// Year-specific suggestions
		if year == "" {
			suggestions = append(suggestions, "Try year='1997' (Constitution era), '1964' (Civil Code), or '2018' (recent EU compliance)")
		}

		// Add cache-enhanced suggestions
		cachedSuggestions := s.getSearchSuggestions(title)
		suggestions = append(suggestions, cachedSuggestions...)

		// Add keyword-based suggestions
		if title != "" {
			keywordSuggestions := s.validateKeywords(title)
			if len(keywordSuggestions) > 0 {
				suggestions = append(suggestions, "")
				suggestions = append(suggestions, "Related legal keywords:")
				suggestions = append(suggestions, keywordSuggestions...)
			}
		}

		// Add search scope recommendations
		suggestions = append(suggestions, "")
		suggestions = append(suggestions, "Search scope tips:")
		suggestions = append(suggestions, "• Add inForce='1' to search only active legislation")
		suggestions = append(suggestions, "• Try publisher='DU' for major laws, 'MP' for regulations")
		suggestions = append(suggestions, "• Use broader date ranges (e.g., year='2020' instead of specific dates)")

		response := StandardResponse{
			Operation:   "Legal Acts Search",
			Status:      "No Results Found",
			Summary:     criteria,
			NextActions: suggestions,
			Note:        "The Polish legal database contains over 160,000 documents. Try broader search terms, check spelling of Polish legal terms, or use the popular searches above.",
		}
		return mcp.NewToolResultText(response.Format()), nil
	}

	// Build results data
	var results []string
	displayCount := 10
	if len(searchResult.Items) < displayCount {
		displayCount = len(searchResult.Items)
	}

	results = append(results, fmt.Sprintf("Showing first %d of %d legal acts:", displayCount, searchResult.Count))

	for i, act := range searchResult.Items {
		if i >= 10 { // Show only first 10 to save space
			break
		}

		title := "No title"
		if act.Title != nil {
			title = *act.Title
		}

		publisher := "Unknown"
		if act.Publisher != nil {
			publisher = *act.Publisher
		}

		year := "Unknown"
		pos := "Unknown"
		if act.Year != nil {
			year = fmt.Sprintf("%d", *act.Year)
		}
		if act.Pos != nil {
			pos = fmt.Sprintf("%d", *act.Pos)
		}

		status := "Unknown"
		if act.InForce != nil {
			switch *act.InForce {
			case "IN_FORCE":
				status = "In force"
			case "NOT_IN_FORCE":
				status = "Not in force"
			default:
				status = "Unknown status"
			}
		}

		results = append(results, fmt.Sprintf("• %s/%s/%s: %s (%s)", publisher, year, pos, title, status))
	}

	if searchResult.Count > 10 {
		results = append(results, fmt.Sprintf("... and %d more acts available", searchResult.Count-10))
	}

	response := StandardResponse{
		Operation: "Legal Acts Search",
		Status:    "Search Completed Successfully",
		Summary:   criteria,
		Data:      results,
		NextActions: buildCrossReferenceHints(searchResult.Items, append([]string{
			"Use eli_get_act_details with publisher/year/position to get full metadata",
			"Use eli_get_act_text to download complete legal text",
			"Use eli_get_act_references to explore legal relationships",
		}, buildPaginationHints(offset, limit, searchResult.Count)...)),
		Note: fmt.Sprintf("Data retrieved from Polish ELI system on %s. Legal acts are continuously updated as new legislation is published.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetActDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")
	position := request.GetString("position", "")
	detailed := request.GetString("detailed", "false")

	if publisher == "" || year == "" || position == "" {
		return mcp.NewToolResultError("All three parameters are required: publisher (e.g., 'DU'), year (e.g., '1997'), and position (e.g., '78'). These identify the exact legal act in the Polish legal system. You can get these values from eli_search_acts results or legal citations."), nil
	}

	// Validate basic format
	if len(year) != 4 {
		return mcp.NewToolResultError(fmt.Sprintf("Year must be a 4-digit year (e.g., '1997', '2020'), but got '%s'.", year)), nil
	}

	endpoint := fmt.Sprintf("%s/acts/%s/%s/%s", eliBaseURL, publisher, year, position)
	apiData, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve legal act details from ELI database: %v. Please verify the legal act coordinates: publisher=%s, year=%s, position=%s. You can search for valid acts using eli_search_acts.", err, publisher, year, position)), nil
	}

	var act eli.Act
	if err := json.Unmarshal(apiData, &act); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse legal act data from ELI API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Build act summary information
	var summary []string
	summary = append(summary, fmt.Sprintf("Act ID: %s/%s/%s", publisher, year, position))

	if act.Title != nil {
		summary = append(summary, fmt.Sprintf("Title: %s", *act.Title))
	}
	if act.Type != nil {
		summary = append(summary, fmt.Sprintf("Type: %s", *act.Type))
	}
	if act.InForce != nil {
		var status string
		switch *act.InForce {
		case "IN_FORCE":
			status = "✓ In force"
		case "NOT_IN_FORCE":
			status = "✗ Not in force"
		default:
			status = "Unknown status"
		}
		summary = append(summary, fmt.Sprintf("Status: %s", status))
	}
	if act.Promulgation != nil {
		summary = append(summary, fmt.Sprintf("Promulgation Date: %s", act.Promulgation.String()))
	}
	if act.EntryIntoForce != nil {
		summary = append(summary, fmt.Sprintf("Entry Into Force: %s", act.EntryIntoForce.String()))
	}

	// Add format availability information
	var data []string
	htmlAvailable := act.TextHTML != nil && *act.TextHTML
	pdfAvailable := act.TextPDF != nil && *act.TextPDF

	data = append(data, "Text Format Availability:")
	if htmlAvailable && pdfAvailable {
		data = append(data, "• HTML format: ✓ Available")
		data = append(data, "• PDF format: ✓ Available")
	} else if htmlAvailable {
		data = append(data, "• HTML format: ✓ Available")
		data = append(data, "• PDF format: ✗ Not available")
	} else if pdfAvailable {
		data = append(data, "• HTML format: ✗ Not available")
		data = append(data, "• PDF format: ✓ Available")
	} else {
		data = append(data, "• HTML format: ✗ Not available")
		data = append(data, "• PDF format: ✗ Not available")
		data = append(data, "Note: This document may not have downloadable text formats")
	}

	// Add reference summary if available
	if act.References != nil && len(*act.References) > 0 {
		totalRefs := 0
		for _, refList := range *act.References {
			totalRefs += len(refList)
		}
		data = append(data, "", fmt.Sprintf("Legal References: %d relationships found across %d categories", totalRefs, len(*act.References)))
	}

	// Build next actions
	var nextActions []string
	if htmlAvailable || pdfAvailable {
		action := fmt.Sprintf("Get full text: eli_get_act_text with publisher='%s', year='%s', position='%s'", publisher, year, position)
		if htmlAvailable && pdfAvailable {
			action += " (both 'html' and 'pdf' formats available)"
		} else if htmlAvailable {
			action += " (use format='html')"
		} else {
			action += " (use format='pdf')"
		}
		nextActions = append(nextActions, action)
	}
	nextActions = append(nextActions,
		fmt.Sprintf("Explore legal relationships: eli_get_act_references with publisher='%s', year='%s', position='%s'", publisher, year, position),
		"Search related acts: eli_search_acts with similar parameters",
		"Get complete metadata: eli_get_act_details with detailed='true'",
	)

	// Return detailed view if requested, otherwise return summary
	if detailed == "true" {
		result, _ := json.MarshalIndent(act, "", "  ")
		data = append(data, "", "Complete legal act metadata including status, dates, institutions, and classification:", "", string(result))

		response := StandardResponse{
			Operation:   "Legal Act Details",
			Status:      "Retrieved Successfully (Detailed View)",
			Summary:     summary,
			Data:        data,
			NextActions: nextActions,
			Note:        fmt.Sprintf("Legal act metadata retrieved on %s. Last legal status update: %s", time.Now().Format("2006-01-02 15:04:05 MST"), getLastUpdateDate(act)),
		}
		return mcp.NewToolResultText(response.Format()), nil
	} else {
		response := StandardResponse{
			Operation:   "Legal Act Details",
			Status:      "Retrieved Successfully",
			Summary:     summary,
			Data:        data,
			NextActions: nextActions,
			Note:        fmt.Sprintf("Summary view retrieved on %s. Last legal status update: %s. Use detailed='true' for complete metadata JSON.", time.Now().Format("2006-01-02 15:04:05 MST"), getLastUpdateDate(act)),
		}
		return mcp.NewToolResultText(response.Format()), nil
	}
}

func (s *SejmServer) handleGetActText(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")
	position := request.GetString("position", "")
	format := request.GetString("format", "html")
	pageStr := request.GetString("page", "")
	pagesPerChunkStr := request.GetString("pages_per_chunk", "5")
	showPageInfo := request.GetString("show_page_info", "false")

	s.logger.Info("eli_get_act_text called",
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("position", position),
		slog.String("format", format),
		slog.String("page", pageStr),
		slog.String("pagesPerChunk", pagesPerChunkStr),
		slog.String("showPageInfo", showPageInfo))

	if publisher == "" || year == "" || position == "" {
		s.logger.Error("Missing required parameters",
			slog.String("publisher", publisher),
			slog.String("year", year),
			slog.String("position", position))
		return mcp.NewToolResultError("All three parameters are required: publisher, year, and position. These identify the exact legal act. Example: publisher='DU', year='1997', position='78' for the Polish Constitution. Get these coordinates from eli_search_acts or eli_get_act_details."), nil
	}

	// Validate format
	if format != "html" && format != "pdf" && format != "text" {
		return mcp.NewToolResultError(fmt.Sprintf("Format must be 'html', 'pdf', or 'text', but got '%s'. HTML is recommended for AI analysis, PDF for official documentation, TEXT for plain text extraction when HTML is unavailable.", format)), nil
	}

	// Check format availability before attempting download
	detailsEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s", eliBaseURL, publisher, year, position)
	detailsData, err := s.makeAPIRequest(ctx, detailsEndpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to verify legal act availability: %v. Please verify the coordinates: publisher=%s, year=%s, position=%s using eli_search_acts first.", err, publisher, year, position)), nil
	}

	var act eli.Act
	if err := json.Unmarshal(detailsData, &act); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse legal act details: %v. Please verify the act exists.", err)), nil
	}

	// Check format availability and provide helpful guidance
	htmlAvailable := act.TextHTML != nil && *act.TextHTML
	pdfAvailable := act.TextPDF != nil && *act.TextPDF

	if format == "html" && !htmlAvailable {
		if pdfAvailable {
			return mcp.NewToolResultError(fmt.Sprintf("HTML format is not available for legal act %s/%s/%s. This document is only available in PDF format. Please retry with format='pdf' to get the document, or format='text' to extract plain text from PDF. Many older legal documents and regulations are only published in PDF format by the Polish legal system.", publisher, year, position)), nil
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("Neither HTML nor PDF format is available for legal act %s/%s/%s. This document may not have downloadable text formats available in the ELI system.", publisher, year, position)), nil
		}
	}

	if format == "pdf" && !pdfAvailable {
		if htmlAvailable {
			return mcp.NewToolResultError(fmt.Sprintf("PDF format is not available for legal act %s/%s/%s. This document is only available in HTML format. Please retry with format='html' to get the structured text.", publisher, year, position)), nil
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("Neither PDF nor HTML format is available for legal act %s/%s/%s. This document may not have downloadable text formats available in the ELI system.", publisher, year, position)), nil
		}
	}

	if format == "text" && !htmlAvailable && !pdfAvailable {
		return mcp.NewToolResultError(fmt.Sprintf("No text formats available for legal act %s/%s/%s. This document does not have HTML or PDF text available for extraction in the ELI system.", publisher, year, position)), nil
	}

	var endpoint string
	var requestFormat string

	s.logger.Info("Format selection",
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("position", position),
		slog.String("requestedFormat", format),
		slog.Bool("htmlAvailable", htmlAvailable),
		slog.Bool("pdfAvailable", pdfAvailable))

	switch format {
	case "text":
		// For text format, handle pagination first, then choose the best available format
		if showPageInfo == "true" || pageStr != "" || pagesPerChunkStr != "" {
			// Pagination requested - must use PDF for page-level control
			if pdfAvailable {
				s.logger.Info("Pagination requested, using PDF extraction route")
				pdfEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s/text.pdf", eliBaseURL, publisher, year, position)
				pdfData, pdfErr := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
				if pdfErr != nil {
					s.logger.Error("Failed to retrieve PDF for pagination", slog.Any("error", pdfErr))
					return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for pagination: %v", pdfErr)), nil
				}

				s.logger.Info("Retrieved PDF data, starting paginated text extraction", slog.Int("bytes", len(pdfData)))
				return s.extractTextWithPagination(ctx, pdfData, publisher, year, position, pageStr, pagesPerChunkStr, showPageInfo)
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("Pagination requested but PDF format not available for legal act %s/%s/%s. Pagination requires PDF format for page-level control.", publisher, year, position)), nil
			}
		}

		// No pagination - use the best available format (prefer HTML for faster processing)
		if htmlAvailable {
			s.logger.Info("Using HTML route for text extraction (no pagination)")
			endpoint = fmt.Sprintf("%s/acts/%s/%s/%s/text.html", eliBaseURL, publisher, year, position)
			requestFormat = "html"
		} else if pdfAvailable {
			s.logger.Info("HTML not available, using direct PDF extraction route")
			// Go directly to PDF extraction since HTML is not available
			pdfEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s/text.pdf", eliBaseURL, publisher, year, position)
			pdfData, pdfErr := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
			if pdfErr != nil {
				s.logger.Error("Failed to retrieve PDF for direct text extraction", slog.Any("error", pdfErr))
				return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for text extraction: %v", pdfErr)), nil
			}

			s.logger.Info("Retrieved PDF data, starting text extraction with pagination support", slog.Int("bytes", len(pdfData)))
			// Extract text from PDF with pagination support
			return s.extractTextWithPagination(ctx, pdfData, publisher, year, position, pageStr, pagesPerChunkStr, showPageInfo)
		} else {
			s.logger.Error("No text formats available",
				slog.String("publisher", publisher),
				slog.String("year", year),
				slog.String("position", position))
			return mcp.NewToolResultError(fmt.Sprintf("No text formats available for legal act %s/%s/%s. This document does not have HTML or PDF text available for extraction.", publisher, year, position)), nil
		}
	case "pdf":
		s.logger.Info("Using PDF format")
		endpoint = fmt.Sprintf("%s/acts/%s/%s/%s/text.pdf", eliBaseURL, publisher, year, position)
		requestFormat = "pdf"
	default:
		s.logger.Info("Using HTML format")
		endpoint = fmt.Sprintf("%s/acts/%s/%s/%s/text.html", eliBaseURL, publisher, year, position)
		requestFormat = "html"
	}

	s.logger.Info("Making text request",
		slog.String("endpoint", endpoint),
		slog.String("format", requestFormat))
	data, err := s.makeTextRequest(ctx, endpoint, requestFormat)
	if err != nil {
		s.logger.Warn("Text request failed", slog.Any("error", err))
		// Special handling for 'text' format - try PDF extraction if HTML fails
		if format == "text" && strings.Contains(err.Error(), "403") {
			s.logger.Info("HTML failed with 403, attempting fallback to PDF extraction")
			// HTML failed, try PDF and extract text
			pdfEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s/text.pdf", eliBaseURL, publisher, year, position)
			pdfData, pdfErr := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
			if pdfErr == nil {
				s.logger.Info("Fallback PDF retrieval successful, starting text extraction with pagination", slog.Int("bytes", len(pdfData)))
				// Extract text from PDF with pagination support
				return s.extractTextWithPagination(ctx, pdfData, publisher, year, position, pageStr, pagesPerChunkStr, showPageInfo)
			} else {
				s.logger.Error("Fallback PDF retrieval also failed", slog.Any("error", pdfErr))
			}
		}

		// If HTML format failed, check if PDF format is available
		if format == "html" && strings.Contains(err.Error(), "403") {
			// Try to get act details to check available formats
			detailsEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s", eliBaseURL, publisher, year, position)
			detailsData, detailsErr := s.makeAPIRequest(ctx, detailsEndpoint, nil)
			if detailsErr == nil {
				var act eli.Act
				if json.Unmarshal(detailsData, &act) == nil {
					// Check if textPDF is available but textHTML is not
					if act.TextPDF != nil && *act.TextPDF && (act.TextHTML == nil || !*act.TextHTML) {
						return mcp.NewToolResultError(fmt.Sprintf("HTML format is not available for legal act %s/%s/%s, but PDF format is available. Please retry with format='pdf' to get the document text, or format='text' to extract plain text from PDF. Some older or special documents are only published in PDF format.", publisher, year, position)), nil
					}
				}
			}
		}

		// Enhanced error messages with specific codes and suggestions
		if strings.Contains(err.Error(), "403") {
			if format == "pdf" {
				return mcp.NewToolResultError(fmt.Sprintf("PDF format access denied (403) for legal act %s/%s/%s. This may indicate: 1) Document not available in PDF format, 2) API access restrictions, or 3) Invalid document coordinates. Try format='html' or verify the act exists using eli_get_act_details first.", publisher, year, position)), nil
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("HTML format access denied (403) for legal act %s/%s/%s. This may indicate: 1) Document not available in HTML format, 2) API access restrictions, or 3) Invalid document coordinates. Try format='pdf' or verify the act exists using eli_get_act_details first.", publisher, year, position)), nil
			}
		} else if strings.Contains(err.Error(), "404") {
			return mcp.NewToolResultError(fmt.Sprintf("Legal act %s/%s/%s not found (404). Please verify the coordinates are correct using eli_search_acts or eli_get_act_details first.", publisher, year, position)), nil
		} else if strings.Contains(err.Error(), "429") {
			return mcp.NewToolResultError("Rate limit exceeded (429). Please wait a moment before trying again. The ELI API has request limits to ensure service availability."), nil
		}

		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve legal act text from ELI database: %v. Please verify the legal act exists with coordinates: publisher=%s, year=%s, position=%s. You can verify existence using eli_get_act_details first.", err, publisher, year, position)), nil
	}

	s.logger.Info("Successfully retrieved text data",
		slog.Int("bytes", len(data)),
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("position", position),
		slog.String("format", format))

	if format == "pdf" {
		s.logger.Info("Returning PDF document", slog.Int("bytes", len(data)))
		return mcp.NewToolResultText(fmt.Sprintf("Successfully retrieved PDF document for legal act %s/%s/%s (%d bytes). This is the official publication-quality version suitable for citations, archival, and formal documentation. The PDF contains the complete legal text as published in the official gazette.", publisher, year, position, len(data))), nil
	}

	if format == "text" {
		s.logger.Info("Returning text extracted from HTML", slog.Int("characters", len(data)))
		// For text format that succeeded via HTML, return the HTML as text
		textSummary := fmt.Sprintf("Successfully retrieved text for legal act %s/%s/%s (%d characters). This text was obtained from the HTML format and is ideal for AI analysis and text processing.", publisher, year, position, len(data))
		textSummary += "\n\n=== LEGAL ACT TEXT BEGINS ==="
		return mcp.NewToolResultText(fmt.Sprintf("%s\n\n%s", textSummary, string(data))), nil
	}

	// For HTML, provide context about the structured content
	textSummary := fmt.Sprintf("Successfully retrieved HTML text for legal act %s/%s/%s (%d characters). This structured format is ideal for AI analysis, text processing, and automated legal research. The content includes:", publisher, year, position, len(data))
	textSummary += "\n- Complete legal text with original structure"
	textSummary += "\n- Article and chapter organization"
	textSummary += "\n- Official legal language and terminology"
	textSummary += "\n- Amendment annotations and references"
	textSummary += "\n\n=== LEGAL ACT TEXT BEGINS ==="

	return mcp.NewToolResultText(fmt.Sprintf("%s\n\n%s", textSummary, string(data))), nil
}

func (s *SejmServer) handleGetActReferences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")
	position := request.GetString("position", "")

	if publisher == "" || year == "" || position == "" {
		return mcp.NewToolResultError("All three parameters are required: publisher, year, and position. These identify the source legal act whose legal relationships you want to explore. Get these coordinates from eli_search_acts or legal citations."), nil
	}

	endpoint := fmt.Sprintf("%s/acts/%s/%s/%s/references", eliBaseURL, publisher, year, position)
	apiData, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve legal act references from ELI database: %v. Please verify the legal act exists with coordinates: publisher=%s, year=%s, position=%s. Use eli_get_act_details to verify the act exists first.", err, publisher, year, position)), nil
	}

	var references eli.CustomReferencesDetailsInfo
	if err := json.Unmarshal(apiData, &references); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse legal references data from ELI API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Analyze reference patterns by category
	totalRefs := 0
	for _, refList := range references {
		totalRefs += len(refList)
	}

	// Build summary information
	var summary []string
	summary = append(summary, fmt.Sprintf("Source Act: %s/%s/%s", publisher, year, position))
	summary = append(summary, fmt.Sprintf("Total reference categories: %d", len(references)))
	summary = append(summary, fmt.Sprintf("Total legal relationships found: %d", totalRefs))

	if totalRefs == 0 {
		response := StandardResponse{
			Operation: "Legal Reference Network Analysis",
			Status:    "No References Found",
			Summary:   summary,
			NextActions: []string{
				"Try analyzing a major law (e.g., Constitution, Civil Code) which typically have many references",
				"Use eli_search_acts to find acts that might reference this one",
				"Check if this is a standalone regulation with limited legal relationships",
			},
			Note: "This legal act has no recorded relationships with other acts in the ELI database. This could mean it's a standalone regulation or the reference data hasn't been fully processed yet.",
		}
		return mcp.NewToolResultText(response.Format()), nil
	}

	// Build reference categories and navigation hints
	var data []string
	var nextActions []string

	data = append(data, "Reference Categories:")

	// Prioritize important reference types for navigation
	priorityCategories := map[string]string{
		"Akty uchylające":  "Acts that repeal this law",
		"Akty zmieniające": "Acts that amend this law",
		"Akty uchylone":    "Acts repealed by this law",
		"Akty zmieniane":   "Acts amended by this law",
		"Akty podstawowe":  "Foundational acts this law is based on",
		"Akty wykonawcze":  "Implementing regulations for this law",
	}

	// Show priority categories first with navigation hints
	for category, description := range priorityCategories {
		if refList, exists := references[category]; exists && len(refList) > 0 {
			data = append(data, fmt.Sprintf("• %s: %d references (%s)", category, len(refList), description))

			// Add specific navigation examples
			for i, ref := range refList {
				if i >= 2 { // Show only first 2 per category
					break
				}
				if ref.Act != nil && ref.Act.Title != nil {
					// Extract coordinates for navigation
					actPublisher := "Unknown"
					actYear := "Unknown"
					actPos := "Unknown"

					if ref.Act.Publisher != nil {
						actPublisher = *ref.Act.Publisher
					}
					if ref.Act.Year != nil {
						actYear = fmt.Sprintf("%d", *ref.Act.Year)
					}
					if ref.Act.Pos != nil {
						actPos = fmt.Sprintf("%d", *ref.Act.Pos)
					}

					data = append(data, fmt.Sprintf("  → %s (%s/%s/%s)", *ref.Act.Title, actPublisher, actYear, actPos))

					// Add to next actions with specific navigation commands
					nextActions = append(nextActions, fmt.Sprintf("Explore '%s': eli_get_act_details with publisher='%s', year='%s', position='%s'", *ref.Act.Title, actPublisher, actYear, actPos))
				}
			}

			if len(refList) > 2 {
				data = append(data, fmt.Sprintf("  ... and %d more in this category", len(refList)-2))
			}
			data = append(data, "") // Add spacing
		}
	}

	// Show remaining categories
	for category, refList := range references {
		if _, isPriority := priorityCategories[category]; !isPriority && len(refList) > 0 {
			data = append(data, fmt.Sprintf("• %s: %d references", category, len(refList)))
		}
	}

	// Add general navigation actions
	nextActions = append(nextActions,
		fmt.Sprintf("Get full details of this act: eli_get_act_details with publisher='%s', year='%s', position='%s'", publisher, year, position),
		fmt.Sprintf("Download text of this act: eli_get_act_text with publisher='%s', year='%s', position='%s'", publisher, year, position),
		"Search for acts by similar topics: eli_search_acts with relevant keywords",
	)

	response := StandardResponse{
		Operation:   "Legal Reference Network Analysis",
		Status:      "Analysis Completed Successfully",
		Summary:     summary,
		Data:        data,
		NextActions: nextActions,
		Note:        "Use the specific navigation commands above to explore related legal documents. Each reference shows exact coordinates for precise document lookup.",
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetPublishers(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	endpoint := fmt.Sprintf("%s/acts", eliBaseURL)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve publishers directory from ELI database: %v. Please try again.", err)), nil
	}

	var publishers []eli.PublishingHouse
	if err := json.Unmarshal(data, &publishers); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse publishers data from ELI API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Analyze publisher landscape
	totalDocuments := 0
	for _, pub := range publishers {
		if pub.ActsCount != nil {
			totalDocuments += int(*pub.ActsCount)
		}
	}

	publisherSummary := "Polish Legal Publishing System Directory:"
	publisherSummary += fmt.Sprintf("\n- Total publishers in system: %d", len(publishers))
	publisherSummary += fmt.Sprintf("\n- Total legal documents available: %d", totalDocuments)
	publisherSummary += "\n\nKey Publishers:"
	publisherSummary += "\n- DU (Dziennik Ustaw): Primary journal for laws, Constitution, major legislation"
	publisherSummary += "\n- MP (Monitor Polski): Secondary legislation, administrative acts, government announcements"
	publisherSummary += "\n- Ministry codes: Department-specific regulations and directives"
	publisherSummary += "\n- Regional codes: Voivodeship and local government publications"

	publisherSummary += "\n\nUse publisher codes in other ELI tools:"
	publisherSummary += "\n- eli_search_acts: Filter by publisher to find specific types of legislation"
	publisherSummary += "\n- eli_get_act_details: Specify exact publisher for precise document identification"
	publisherSummary += "\n- eli_get_act_text: Retrieve documents from specific publishing authorities"

	result, _ := json.MarshalIndent(publishers, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%s\n\nComplete publishers directory with codes, names, document counts, and operational details:\n\n%s", publisherSummary, string(result))), nil
}

func (s *SejmServer) handleSearchActContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")
	position := request.GetString("position", "")
	searchTerms := request.GetString("search_terms", "")
	contextChars := request.GetString("context_chars", "100")
	maxMatchesPerTerm := request.GetString("max_matches_per_term", "10")

	s.logger.Info("eli_search_act_content called",
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("position", position),
		slog.String("searchTerms", searchTerms),
		slog.String("contextChars", contextChars),
		slog.String("maxMatchesPerTerm", maxMatchesPerTerm))

	if publisher == "" || year == "" || position == "" {
		return mcp.NewToolResultError("All three parameters are required: publisher, year, and position. These identify the exact legal act to search within."), nil
	}

	if searchTerms == "" {
		return mcp.NewToolResultError("Search terms are required. Provide comma-separated terms to search for (e.g., 'artykuł,konstytucja,prawa' or 'podatek,VAT')."), nil
	}

	// Parse parameters
	contextCharsInt := 100
	if contextChars != "" {
		if parsed, err := fmt.Sscanf(contextChars, "%d", &contextCharsInt); parsed == 1 && err == nil {
			if contextCharsInt > 500 {
				contextCharsInt = 500 // max limit
			} else if contextCharsInt < 20 {
				contextCharsInt = 20 // min limit
			}
		} else {
			contextCharsInt = 100 // fallback
		}
	}

	maxMatchesInt := 10
	if maxMatchesPerTerm != "" {
		if parsed, err := fmt.Sscanf(maxMatchesPerTerm, "%d", &maxMatchesInt); parsed == 1 && err == nil {
			if maxMatchesInt > 50 {
				maxMatchesInt = 50 // max limit
			} else if maxMatchesInt < 1 {
				maxMatchesInt = 1 // min limit
			}
		} else {
			maxMatchesInt = 10 // fallback
		}
	}

	// Split and clean search terms
	terms := strings.Split(searchTerms, ",")
	var cleanTerms []string
	for _, term := range terms {
		cleaned := strings.TrimSpace(term)
		if cleaned != "" {
			cleanTerms = append(cleanTerms, cleaned)
		}
	}

	if len(cleanTerms) == 0 {
		return mcp.NewToolResultError("No valid search terms found. Please provide comma-separated terms to search for."), nil
	}

	// First, get the PDF to extract text page by page
	pdfEndpoint := fmt.Sprintf("%s/acts/%s/%s/%s/text.pdf", eliBaseURL, publisher, year, position)
	pdfData, err := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for search: %v. Please verify the legal act coordinates: publisher=%s, year=%s, position=%s", err, publisher, year, position)), nil
	}

	s.logger.Info("Retrieved PDF for content search", slog.Int("bytes", len(pdfData)))

	// Parse PDF and extract text page by page
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		s.logger.Error("Failed to parse PDF for content search", slog.Any("error", err))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse PDF document: %v", err)), nil
	}
	defer func() {
		if err := doc.Close(); err != nil {
			s.logger.Warn("Failed to close PDF document", slog.Any("error", err))
		}
	}()

	pageCount := doc.NumPage()
	s.logger.Info("PDF parsed for content search", slog.Int("totalPages", pageCount))

	if pageCount == 0 {
		return mcp.NewToolResultError("PDF document has no pages to search"), nil
	}

	// Search results structure
	type SearchMatch struct {
		Term     string
		Page     int
		Context  string
		Position int // character position within page
	}

	termMatches := make(map[string][]SearchMatch)
	totalMatches := 0

	// Search each page
	for pageNum := 0; pageNum < pageCount; pageNum++ {
		pageText, err := doc.Text(pageNum)
		if err != nil {
			s.logger.Warn("Failed to extract text from page for search",
				slog.Int("page", pageNum+1), slog.Any("error", err))
			continue
		}

		pageTextLower := strings.ToLower(pageText)

		// Search for each term on this page
		for _, term := range cleanTerms {
			termLower := strings.ToLower(term)

			// Skip if we already have enough matches for this term
			if len(termMatches[term]) >= maxMatchesInt {
				continue
			}

			// Find all occurrences of this term on this page
			startPos := 0
			for {
				pos := strings.Index(pageTextLower[startPos:], termLower)
				if pos == -1 {
					break
				}

				actualPos := startPos + pos

				// Extract context around the match
				contextStart := actualPos - contextCharsInt/2
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := actualPos + len(term) + contextCharsInt/2
				if contextEnd > len(pageText) {
					contextEnd = len(pageText)
				}

				context := pageText[contextStart:contextEnd]
				// Highlight the found term in context
				context = strings.ReplaceAll(context, pageText[actualPos:actualPos+len(term)],
					fmt.Sprintf("**%s**", pageText[actualPos:actualPos+len(term)]))

				// Clean up context (remove excessive whitespace)
				context = strings.ReplaceAll(context, "\n", " ")
				context = strings.ReplaceAll(context, "\t", " ")
				for strings.Contains(context, "  ") {
					context = strings.ReplaceAll(context, "  ", " ")
				}
				context = strings.TrimSpace(context)

				match := SearchMatch{
					Term:     term,
					Page:     pageNum + 1, // Convert to 1-based
					Context:  context,
					Position: actualPos,
				}

				termMatches[term] = append(termMatches[term], match)
				totalMatches++

				// Check if we have enough matches for this term
				if len(termMatches[term]) >= maxMatchesInt {
					break
				}

				// Move past this match to find next occurrence
				startPos = actualPos + len(term)
			}
		}
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Document searched: %s/%s/%s", publisher, year, position))
	summary = append(summary, fmt.Sprintf("Search terms: %d (%s)", len(cleanTerms), strings.Join(cleanTerms, ", ")))
	summary = append(summary, fmt.Sprintf("Total pages searched: %d", pageCount))
	summary = append(summary, fmt.Sprintf("Total matches found: %d", totalMatches))

	var data []string
	var nextActions []string

	if totalMatches == 0 {
		data = append(data, "No matches found for any search terms.")
		data = append(data, "")
		data = append(data, "Search suggestions:")
		data = append(data, "• Try broader terms or synonyms")
		data = append(data, "• Check spelling of Polish legal terms")
		data = append(data, "• Use partial words (e.g., 'konstytuc' instead of 'konstytucja')")
		data = append(data, "• Try searching for article numbers (e.g., 'art. 15', 'artykuł 20')")
		data = append(data, "• Use eli_get_keywords to discover available legal keywords")

		nextActions = append(nextActions, "Try different search terms")
		nextActions = append(nextActions, "Use eli_get_keywords for legal terminology suggestions")
		nextActions = append(nextActions, "Use eli_get_act_text with show_page_info='true' to explore document structure")
		nextActions = append(nextActions, "Search for common legal terms like 'artykuł', 'ustęp', 'punkt'")
	} else {
		data = append(data, "Search Results by Term:")
		data = append(data, "")

		// Show results for each term
		for _, term := range cleanTerms {
			matches := termMatches[term]
			if len(matches) > 0 {
				data = append(data, fmt.Sprintf("🔍 '%s' - %d matches:", term, len(matches)))

				// Group by page for cleaner display
				pageGroups := make(map[int][]SearchMatch)
				for _, match := range matches {
					pageGroups[match.Page] = append(pageGroups[match.Page], match)
				}

				var pages []int
				for page := range pageGroups {
					pages = append(pages, page)
				}
				// Sort pages
				for i := 0; i < len(pages)-1; i++ {
					for j := i + 1; j < len(pages); j++ {
						if pages[i] > pages[j] {
							pages[i], pages[j] = pages[j], pages[i]
						}
					}
				}

				for _, page := range pages {
					pageMatches := pageGroups[page]
					data = append(data, fmt.Sprintf("  📄 Page %d (%d matches):", page, len(pageMatches)))

					// Show first few matches from this page
					showCount := len(pageMatches)
					if showCount > 3 {
						showCount = 3 // Limit per page to save space
					}

					for i := 0; i < showCount; i++ {
						match := pageMatches[i]
						data = append(data, fmt.Sprintf("    • %s", match.Context))
					}

					if len(pageMatches) > showCount {
						data = append(data, fmt.Sprintf("    ... and %d more matches on this page", len(pageMatches)-showCount))
					}
				}

				// Add navigation action for this term
				if len(matches) > 0 {
					firstPage := matches[0].Page
					nextActions = append(nextActions, fmt.Sprintf("Read page %d: eli_get_act_text with page='%d' (contains '%s')", firstPage, firstPage, term))
				}

				data = append(data, "")
			} else {
				data = append(data, fmt.Sprintf("❌ '%s' - no matches found", term))
			}
		}

		// Add general navigation actions
		nextActions = append(nextActions, "Use eli_get_act_text with specific page numbers to read full context")
		nextActions = append(nextActions, "Search for related terms to find more relevant sections")
		nextActions = append(nextActions, "Use eli_get_act_references to explore related legal documents")
	}

	response := StandardResponse{
		Operation:   "Legal Act Content Search",
		Status:      "Search Completed Successfully",
		Summary:     summary,
		Data:        data,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Searched %d pages with %d characters context per match. Found %d total matches across %d search terms.", pageCount, contextCharsInt, totalMatches, len(cleanTerms)),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

// extractTextFromPDF extracts plain text from PDF data using go-fitz
func (s *SejmServer) extractTextFromPDF(pdfData []byte) (string, error) {
	s.logger.Info("Starting PDF text extraction", slog.Int("bytes", len(pdfData)))

	if len(pdfData) == 0 {
		s.logger.Error("PDF data is empty")
		return "", fmt.Errorf("PDF data is empty")
	}

	// Create a document from PDF bytes
	s.logger.Debug("Parsing PDF document with go-fitz")
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		s.logger.Error("Failed to parse PDF document",
			slog.Int("bytes", len(pdfData)),
			slog.Any("error", err))
		return "", fmt.Errorf("failed to parse PDF document (%d bytes): %w", len(pdfData), err)
	}
	defer func() {
		if err := doc.Close(); err != nil {
			s.logger.Warn("Failed to close PDF document", slog.Any("error", err))
		}
	}()

	pageCount := doc.NumPage()
	s.logger.Info("PDF document parsed", slog.Int("pages", pageCount))
	if pageCount == 0 {
		s.logger.Error("PDF document has no pages")
		return "", fmt.Errorf("PDF document has no pages")
	}

	var textBuilder strings.Builder
	var extractedPages int
	var failedPages int

	// Extract text from each page
	for i := 0; i < pageCount; i++ {
		s.logger.Debug("Extracting text from page",
			slog.Int("page", i+1),
			slog.Int("totalPages", pageCount))
		text, err := doc.Text(i)
		if err != nil {
			s.logger.Warn("Failed to extract text from page",
				slog.Int("page", i+1),
				slog.Any("error", err))
			failedPages++
			// Don't fail completely, just log and continue with other pages
			continue
		}

		textLength := len(strings.TrimSpace(text))
		if textLength > 0 {
			textBuilder.WriteString(text)
			textBuilder.WriteString("\n\n") // Add page break
			extractedPages++
			s.logger.Debug("Extracted text from page",
				slog.Int("page", i+1),
				slog.Int("characters", textLength))
		} else {
			s.logger.Debug("Page contains no extractable text", slog.Int("page", i+1))
		}
	}

	extractedText := textBuilder.String()

	// Basic cleanup - remove excessive whitespace
	extractedText = strings.TrimSpace(extractedText)

	s.logger.Info("PDF text extraction completed",
		slog.Int("totalPages", pageCount),
		slog.Int("successfulPages", extractedPages),
		slog.Int("failedPages", failedPages),
		slog.Int("totalCharacters", len(extractedText)))

	if len(extractedText) == 0 {
		s.logger.Error("No text could be extracted from PDF document",
			slog.Int("pages", pageCount),
			slog.Int("extractablePages", extractedPages))
		return "", fmt.Errorf("no text could be extracted from PDF document (%d pages, %d with extractable text)", pageCount, extractedPages)
	}

	return extractedText, nil
}

// extractTextWithPagination extracts text from PDF with pagination support
func (s *SejmServer) extractTextWithPagination(ctx context.Context, pdfData []byte, publisher, year, position, pageStr, pagesPerChunkStr, showPageInfo string) (*mcp.CallToolResult, error) {
	s.logger.Info("Starting paginated PDF text extraction",
		slog.Int("bytes", len(pdfData)),
		slog.String("publisher", publisher),
		slog.String("year", year),
		slog.String("position", position),
		slog.String("page", pageStr),
		slog.String("pagesPerChunk", pagesPerChunkStr),
		slog.String("showPageInfo", showPageInfo))

	if len(pdfData) == 0 {
		s.logger.Error("PDF data is empty for pagination")
		return mcp.NewToolResultError("PDF data is empty"), nil
	}

	// Create a document from PDF bytes
	s.logger.Debug("Parsing PDF document for pagination")
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		s.logger.Error("Failed to parse PDF document for pagination",
			slog.Int("bytes", len(pdfData)),
			slog.Any("error", err))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse PDF document (%d bytes): %v", len(pdfData), err)), nil
	}
	defer func() {
		if err := doc.Close(); err != nil {
			s.logger.Warn("Failed to close PDF document", slog.Any("error", err))
		}
	}()

	pageCount := doc.NumPage()
	s.logger.Info("PDF document parsed for pagination", slog.Int("totalPages", pageCount))

	if pageCount == 0 {
		s.logger.Error("PDF document has no pages for pagination")
		return mcp.NewToolResultError("PDF document has no pages"), nil
	}

	// Parse pagination parameters
	pagesPerChunk := 5 // default
	if pagesPerChunkStr != "" {
		if parsed, parseErr := fmt.Sscanf(pagesPerChunkStr, "%d", &pagesPerChunk); parsed == 1 && parseErr == nil {
			if pagesPerChunk > 20 {
				pagesPerChunk = 20 // max limit
			} else if pagesPerChunk < 1 {
				pagesPerChunk = 1 // min limit
			}
		} else {
			pagesPerChunk = 5 // fallback to default
		}
	}

	// Handle show_page_info request
	if showPageInfo == "true" {
		response := StandardResponse{
			Operation: "PDF Page Information",
			Status:    "Retrieved Successfully",
			Summary: []string{
				fmt.Sprintf("Document: %s/%s/%s", publisher, year, position),
				fmt.Sprintf("Total pages: %d", pageCount),
				fmt.Sprintf("Default pages per chunk: %d", pagesPerChunk),
			},
			Data: []string{
				"Page Navigation Instructions:",
				"• Use page='1' to start from first page",
				fmt.Sprintf("• Use pages_per_chunk='%d' to get %d pages at once (max: 20)", pagesPerChunk, pagesPerChunk),
				fmt.Sprintf("• Page ranges: 1-%d available", pageCount),
				"",
				"Examples:",
				"• eli_get_act_text with page='1' and pages_per_chunk='3' (gets pages 1-3)",
				"• eli_get_act_text with page='5' and pages_per_chunk='5' (gets pages 5-9)",
				fmt.Sprintf("• eli_get_act_text with page='%d' (gets last page)", pageCount),
			},
			NextActions: []string{
				"Read specific pages: specify page and pages_per_chunk parameters",
				"Read full document: use eli_get_act_text without pagination parameters",
				"Navigate through document: use sequential page numbers",
			},
			Note: fmt.Sprintf("This document has %d pages. Use pagination to avoid large token responses and read efficiently.", pageCount),
		}
		return mcp.NewToolResultText(response.Format()), nil
	}

	// Parse page parameter
	startPage := 1 // default to first page
	if pageStr != "" {
		if parsed, parseErr := fmt.Sscanf(pageStr, "%d", &startPage); parsed == 1 && parseErr == nil {
			if startPage < 1 {
				startPage = 1
			} else if startPage > pageCount {
				return mcp.NewToolResultError(fmt.Sprintf("Page %d is out of range. Document has only %d pages. Use page numbers 1-%d.", startPage, pageCount, pageCount)), nil
			}
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid page number '%s'. Please use a number between 1 and %d.", pageStr, pageCount)), nil
		}
	}

	// Calculate page range
	endPage := startPage + pagesPerChunk - 1
	if endPage > pageCount {
		endPage = pageCount
	}

	s.logger.Info("Extracting text from page range",
		slog.Int("startPage", startPage),
		slog.Int("endPage", endPage),
		slog.Int("totalPages", pageCount),
		slog.Int("pagesPerChunk", pagesPerChunk))

	// Extract text from specified page range
	var textBuilder strings.Builder
	var extractedPages int
	var failedPages int

	for pageNum := startPage - 1; pageNum < endPage; pageNum++ { // Convert to 0-based indexing
		s.logger.Debug("Extracting text from page",
			slog.Int("page", pageNum+1),
			slog.Int("totalPages", pageCount))

		text, err := doc.Text(pageNum)
		if err != nil {
			s.logger.Warn("Failed to extract text from page",
				slog.Int("page", pageNum+1),
				slog.Any("error", err))
			failedPages++
			continue
		}

		textLength := len(strings.TrimSpace(text))
		if textLength > 0 {
			if extractedPages > 0 {
				textBuilder.WriteString("\n\n--- Page ")
				textBuilder.WriteString(fmt.Sprintf("%d", pageNum+1))
				textBuilder.WriteString(" ---\n\n")
			}
			textBuilder.WriteString(text)
			extractedPages++
			s.logger.Debug("Extracted text from page",
				slog.Int("page", pageNum+1),
				slog.Int("characters", textLength))
		} else {
			s.logger.Debug("Page contains no extractable text", slog.Int("page", pageNum+1))
		}
	}

	extractedText := textBuilder.String()
	extractedText = strings.TrimSpace(extractedText)

	s.logger.Info("Paginated PDF text extraction completed",
		slog.Int("requestedPages", endPage-startPage+1),
		slog.Int("successfulPages", extractedPages),
		slog.Int("failedPages", failedPages),
		slog.Int("totalCharacters", len(extractedText)))

	if len(extractedText) == 0 {
		s.logger.Error("No text could be extracted from requested page range",
			slog.Int("startPage", startPage),
			slog.Int("endPage", endPage),
			slog.Int("extractablePages", extractedPages))
		return mcp.NewToolResultError(fmt.Sprintf("No text could be extracted from pages %d-%d (%d pages, %d with extractable text)", startPage, endPage, endPage-startPage+1, extractedPages)), nil
	}

	// Build response with navigation information
	var summary []string
	summary = append(summary, fmt.Sprintf("Document: %s/%s/%s", publisher, year, position))
	summary = append(summary, fmt.Sprintf("Pages extracted: %d-%d of %d total pages", startPage, endPage, pageCount))
	summary = append(summary, fmt.Sprintf("Successfully extracted: %d pages", extractedPages))
	if failedPages > 0 {
		summary = append(summary, fmt.Sprintf("Failed to extract: %d pages", failedPages))
	}
	summary = append(summary, fmt.Sprintf("Text length: %d characters", len(extractedText)))

	var nextActions []string
	if endPage < pageCount {
		nextPageStart := endPage + 1
		nextPageEnd := nextPageStart + pagesPerChunk - 1
		if nextPageEnd > pageCount {
			nextPageEnd = pageCount
		}
		nextActions = append(nextActions, fmt.Sprintf("Read next pages: eli_get_act_text with page='%d' and pages_per_chunk='%d' (pages %d-%d)", nextPageStart, pagesPerChunk, nextPageStart, nextPageEnd))
	}
	if startPage > 1 {
		prevPageStart := startPage - pagesPerChunk
		if prevPageStart < 1 {
			prevPageStart = 1
		}
		nextActions = append(nextActions, fmt.Sprintf("Read previous pages: eli_get_act_text with page='%d' and pages_per_chunk='%d'", prevPageStart, pagesPerChunk))
	}
	nextActions = append(nextActions, "Get page information: eli_get_act_text with show_page_info='true'")
	nextActions = append(nextActions, "Read full document: eli_get_act_text without pagination parameters")

	response := StandardResponse{
		Operation: "Legal Act Text (Paginated)",
		Status:    "Retrieved Successfully",
		Summary:   summary,
		Data: []string{
			fmt.Sprintf("=== LEGAL ACT TEXT - PAGES %d-%d ===", startPage, endPage),
			"",
			extractedText,
		},
		NextActions: nextActions,
		Note:        fmt.Sprintf("Showing pages %d-%d of %d. Use pagination parameters to navigate through the document efficiently and avoid large token responses.", startPage, endPage, pageCount),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

// getLastUpdateDate extracts the most recent update information from an act
func getLastUpdateDate(act eli.Act) string {
	if act.ChangeDate != nil {
		return act.ChangeDate.String()
	}
	if act.LegalStatusDate != nil {
		return act.LegalStatusDate.String()
	}
	if act.Promulgation != nil {
		return act.Promulgation.String()
	}
	return "Unknown"
}

// searchPDFContent is a generic function to search within PDF documents and return page locations
func (s *SejmServer) searchPDFContent(ctx context.Context, pdfData []byte, documentName, searchTerms string, contextCharsInt, maxMatchesInt int) (*mcp.CallToolResult, error) {
	s.logger.Info("Starting PDF content search",
		slog.String("document", documentName),
		slog.String("searchTerms", searchTerms),
		slog.Int("contextChars", contextCharsInt),
		slog.Int("maxMatches", maxMatchesInt),
		slog.Int("pdfBytes", len(pdfData)))

	if len(pdfData) == 0 {
		return mcp.NewToolResultError("PDF data is empty"), nil
	}

	// Parse PDF and extract text page by page
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		s.logger.Error("Failed to parse PDF for content search", slog.Any("error", err))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse PDF document: %v", err)), nil
	}
	defer func() {
		if err := doc.Close(); err != nil {
			s.logger.Warn("Failed to close PDF document", slog.Any("error", err))
		}
	}()

	pageCount := doc.NumPage()
	s.logger.Info("PDF parsed for content search", slog.Int("totalPages", pageCount))

	if pageCount == 0 {
		return mcp.NewToolResultError("PDF document has no pages to search"), nil
	}

	// Split and clean search terms
	terms := strings.Split(searchTerms, ",")
	var cleanTerms []string
	for _, term := range terms {
		cleaned := strings.TrimSpace(term)
		if cleaned != "" {
			cleanTerms = append(cleanTerms, cleaned)
		}
	}

	if len(cleanTerms) == 0 {
		return mcp.NewToolResultError("No valid search terms found. Please provide comma-separated terms to search for."), nil
	}

	// Search results structure
	type SearchMatch struct {
		Term     string
		Page     int
		Context  string
		Position int // character position within page
	}

	termMatches := make(map[string][]SearchMatch)
	totalMatches := 0

	// Search each page
	for pageNum := 0; pageNum < pageCount; pageNum++ {
		pageText, err := doc.Text(pageNum)
		if err != nil {
			s.logger.Warn("Failed to extract text from page for search",
				slog.Int("page", pageNum+1), slog.Any("error", err))
			continue
		}

		pageTextLower := strings.ToLower(pageText)

		// Search for each term on this page
		for _, term := range cleanTerms {
			termLower := strings.ToLower(term)

			// Skip if we already have enough matches for this term
			if len(termMatches[term]) >= maxMatchesInt {
				continue
			}

			// Find all occurrences of this term on this page
			startPos := 0
			for {
				pos := strings.Index(pageTextLower[startPos:], termLower)
				if pos == -1 {
					break
				}

				actualPos := startPos + pos

				// Extract context around the match
				contextStart := actualPos - contextCharsInt/2
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := actualPos + len(term) + contextCharsInt/2
				if contextEnd > len(pageText) {
					contextEnd = len(pageText)
				}

				context := pageText[contextStart:contextEnd]
				// Highlight the found term in context
				context = strings.ReplaceAll(context, pageText[actualPos:actualPos+len(term)],
					fmt.Sprintf("**%s**", pageText[actualPos:actualPos+len(term)]))

				// Clean up context (remove excessive whitespace)
				context = strings.ReplaceAll(context, "\n", " ")
				context = strings.ReplaceAll(context, "\t", " ")
				for strings.Contains(context, "  ") {
					context = strings.ReplaceAll(context, "  ", " ")
				}
				context = strings.TrimSpace(context)

				match := SearchMatch{
					Term:     term,
					Page:     pageNum + 1, // Convert to 1-based
					Context:  context,
					Position: actualPos,
				}

				termMatches[term] = append(termMatches[term], match)
				totalMatches++

				// Check if we have enough matches for this term
				if len(termMatches[term]) >= maxMatchesInt {
					break
				}

				// Move past this match to find next occurrence
				startPos = actualPos + len(term)
			}
		}
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Document searched: %s", documentName))
	summary = append(summary, fmt.Sprintf("Search terms: %d (%s)", len(cleanTerms), strings.Join(cleanTerms, ", ")))
	summary = append(summary, fmt.Sprintf("Total pages searched: %d", pageCount))
	summary = append(summary, fmt.Sprintf("Total matches found: %d", totalMatches))

	var data []string
	var nextActions []string

	if totalMatches == 0 {
		data = append(data, "No matches found for any search terms.")
		data = append(data, "")
		data = append(data, "Search suggestions:")
		data = append(data, "• Try broader terms or synonyms")
		data = append(data, "• Check spelling of Polish terms")
		data = append(data, "• Use partial words (e.g., 'Kowal' instead of 'Kowalski')")
		data = append(data, "• Try searching for common voting terms like 'za', 'przeciw', 'wstrzymał'")
		data = append(data, "• Use sejm_get_parliamentary_keywords to discover effective search terms")

		nextActions = append(nextActions, "Try different search terms")
		nextActions = append(nextActions, "Use sejm_get_parliamentary_keywords for suggested keywords")
		nextActions = append(nextActions, "Use broader search terms")
	} else {
		data = append(data, "Search Results by Term:")
		data = append(data, "")

		// Show results for each term
		for _, term := range cleanTerms {
			matches := termMatches[term]
			if len(matches) > 0 {
				data = append(data, fmt.Sprintf("🔍 '%s' - %d matches:", term, len(matches)))

				// Group by page for cleaner display
				pageGroups := make(map[int][]SearchMatch)
				for _, match := range matches {
					pageGroups[match.Page] = append(pageGroups[match.Page], match)
				}

				var pages []int
				for page := range pageGroups {
					pages = append(pages, page)
				}
				// Sort pages
				for i := 0; i < len(pages)-1; i++ {
					for j := i + 1; j < len(pages); j++ {
						if pages[i] > pages[j] {
							pages[i], pages[j] = pages[j], pages[i]
						}
					}
				}

				for _, page := range pages {
					pageMatches := pageGroups[page]
					data = append(data, fmt.Sprintf("  📄 Page %d (%d matches):", page, len(pageMatches)))

					// Show first few matches from this page
					showCount := len(pageMatches)
					if showCount > 3 {
						showCount = 3 // Limit per page to save space
					}

					for i := 0; i < showCount; i++ {
						match := pageMatches[i]
						data = append(data, fmt.Sprintf("    • %s", match.Context))
					}

					if len(pageMatches) > showCount {
						data = append(data, fmt.Sprintf("    ... and %d more matches on this page", len(pageMatches)-showCount))
					}
				}

				data = append(data, "")
			} else {
				data = append(data, fmt.Sprintf("❌ '%s' - no matches found", term))
			}
		}

		nextActions = append(nextActions, "Use specific page numbers to read full context")
		nextActions = append(nextActions, "Search for related terms to find more relevant sections")
	}

	response := StandardResponse{
		Operation:   "PDF Content Search",
		Status:      "Search Completed Successfully",
		Summary:     summary,
		Data:        data,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Searched %d pages with %d characters context per match. Found %d total matches across %d search terms.", pageCount, contextCharsInt, totalMatches, len(cleanTerms)),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetKeywords(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_get_keywords called", slog.Any("arguments", request.Params.Arguments))

	// Fetch keywords from ELI API
	endpoint := "https://api.sejm.gov.pl/eli/keywords"
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve keywords: %v", err)), nil
	}

	var keywords []string
	if err := json.Unmarshal(data, &keywords); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse keywords: %v", err)), nil
	}

	// Apply filter if provided
	filter := request.GetString("filter", "")
	if filter != "" {
		var filteredKeywords []string
		filterLower := strings.ToLower(filter)
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(keyword), filterLower) {
				filteredKeywords = append(filteredKeywords, keyword)
			}
		}
		keywords = filteredKeywords
	}

	// Apply sorting
	sortOrder := request.GetString("sort", "asc")
	if sortOrder == "desc" {
		sort.Sort(sort.Reverse(sort.StringSlice(keywords)))
	} else {
		sort.Strings(keywords)
	}

	// Format response
	var summary []string
	if filter != "" {
		summary = append(summary, fmt.Sprintf("Found %d keywords containing '%s'", len(keywords), filter))
	} else {
		summary = append(summary, fmt.Sprintf("Retrieved all %d available legal keywords", len(keywords)))
	}

	keywordsList := strings.Join(keywords, "\n• ")
	formattedData := fmt.Sprintf("Legal Keywords (%d total):\n• %s", len(keywords), keywordsList)

	response := StandardResponse{
		Operation: "ELI Legal Keywords Directory",
		Status:    "Retrieved Successfully",
		Summary:   summary,
		Data:      []string{formattedData},
		NextActions: []string{
			"Use keywords in eli_search_acts parameter: keyword",
			"Combine multiple keywords with commas",
		},
		Note: fmt.Sprintf("Keywords retrieved on %s. Use exact terms for searches.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetTypes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_get_types called", slog.Any("arguments", request.Params.Arguments))

	// Use hardcoded document types (static data that rarely changes)
	types := make([]string, len(eliDocumentTypes))
	copy(types, eliDocumentTypes)

	// Apply filter if provided
	filter := request.GetString("filter", "")
	if filter != "" {
		var filteredTypes []string
		filterLower := strings.ToLower(filter)
		for _, docType := range types {
			if strings.Contains(strings.ToLower(docType), filterLower) {
				filteredTypes = append(filteredTypes, docType)
			}
		}
		types = filteredTypes
	}

	// Apply sorting
	sortOrder := request.GetString("sort", "asc")
	if sortOrder == "desc" {
		sort.Sort(sort.Reverse(sort.StringSlice(types)))
	} else {
		sort.Strings(types)
	}

	// Format response
	var summary []string
	if filter != "" {
		summary = append(summary, fmt.Sprintf("Found %d document types containing '%s'", len(types), filter))
	} else {
		summary = append(summary, fmt.Sprintf("Retrieved all %d available document types", len(types)))
	}

	typesList := strings.Join(types, "\n• ")
	formattedData := fmt.Sprintf("Legal Document Types (%d total):\n• %s", len(types), typesList)

	response := StandardResponse{
		Operation: "ELI Document Types Directory",
		Status:    "Retrieved Successfully",
		Summary:   summary,
		Data:      []string{formattedData},
		NextActions: []string{
			"Use document types in eli_search_acts parameter: type",
			"Common types: 'Ustawa' (law), 'Rozporządzenie' (regulation), 'Dekret' (decree)",
		},
		Note: fmt.Sprintf("Document types retrieved on %s. Use exact terms for type searches.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetStatuses(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_get_statuses called", slog.Any("arguments", request.Params.Arguments))

	// Use hardcoded legal statuses (static data that rarely changes)
	statuses := make([]string, len(eliLegalStatuses))
	copy(statuses, eliLegalStatuses)

	// Apply filter if provided
	filter := request.GetString("filter", "")
	if filter != "" {
		var filteredStatuses []string
		filterLower := strings.ToLower(filter)
		for _, status := range statuses {
			if strings.Contains(strings.ToLower(status), filterLower) {
				filteredStatuses = append(filteredStatuses, status)
			}
		}
		statuses = filteredStatuses
	}

	// Apply sorting
	sortOrder := request.GetString("sort", "asc")
	if sortOrder == "desc" {
		sort.Sort(sort.Reverse(sort.StringSlice(statuses)))
	} else {
		sort.Strings(statuses)
	}

	// Format response
	var summary []string
	if filter != "" {
		summary = append(summary, fmt.Sprintf("Found %d legal statuses containing '%s'", len(statuses), filter))
	} else {
		summary = append(summary, fmt.Sprintf("Retrieved all %d available legal statuses", len(statuses)))
	}

	statusesList := strings.Join(statuses, "\n• ")
	formattedData := fmt.Sprintf("Legal Status Classifications (%d total):\n• %s", len(statuses), statusesList)

	response := StandardResponse{
		Operation: "ELI Legal Statuses Directory",
		Status:    "Retrieved Successfully",
		Summary:   summary,
		Data:      []string{formattedData},
		NextActions: []string{
			"Use statuses for filtering current legal validity",
			"Key statuses: 'obowiązujący' (in force), 'uchylony' (repealed), 'nieobowiązujący' (not in force)",
		},
		Note: fmt.Sprintf("Legal statuses retrieved on %s. Use for compliance and validity checking.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleListActs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_list_acts called", slog.Any("arguments", request.Params.Arguments))

	// Use the search endpoint to list acts with pagination
	params := make(map[string]string)

	limit := request.GetString("limit", "50")
	params["limit"] = limit

	offset := request.GetString("offset", "0")
	if offset != "" {
		params["offset"] = offset
	}

	endpoint := "https://api.sejm.gov.pl/eli/acts/search"
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve acts listing: %v", err)), nil
	}

	var searchResult struct {
		Items []eli.Act `json:"items"`
		Count int       `json:"count"`
	}
	if err := json.Unmarshal(data, &searchResult); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse acts data: %v", err)), nil
	}

	// Create summary
	var summary []string
	var results []string
	var nextActions []string

	summary = append(summary, fmt.Sprintf("Retrieved %d legal acts (limit: %s, offset: %s)", len(searchResult.Items), limit, offset))

	// Show sample acts
	for i, act := range searchResult.Items {
		if i >= 10 { // Show first 10 as sample
			break
		}

		title := "No title"
		if act.Title != nil {
			title = *act.Title
		}

		publisher := "Unknown"
		if act.Publisher != nil {
			publisher = *act.Publisher
		}

		year := "Unknown"
		if act.Year != nil {
			year = fmt.Sprintf("%d", *act.Year)
		}

		results = append(results, fmt.Sprintf("%d. %s (%s %s)", i+1, title, publisher, year))
	}

	if len(searchResult.Items) > 10 {
		results = append(results, fmt.Sprintf("... and %d more acts", len(searchResult.Items)-10))
	}

	// Suggest next actions
	nextActions = append(nextActions, "Use eli_search_acts for targeted searches with specific criteria")
	nextActions = append(nextActions, "Use eli_get_act_details for detailed information about specific acts")
	if len(searchResult.Items) == parseInt(limit) {
		nextActions = append(nextActions, fmt.Sprintf("Continue browsing: use offset='%d' to see more results", parseInt(offset)+parseInt(limit)))
	}

	response := StandardResponse{
		Operation:   "ELI Acts Listing",
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Acts listing retrieved on %s. Use for browsing available legal documents.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetActsByPublisher(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_get_acts_by_publisher called", slog.Any("arguments", request.Params.Arguments))

	publisher := request.GetString("publisher", "")
	if publisher == "" {
		return mcp.NewToolResultError("Publisher parameter is required. Get publisher codes from eli_get_publishers."), nil
	}

	// Use search endpoint to get acts by publisher with pagination
	params := make(map[string]string)
	params["publisher"] = publisher
	limit := request.GetString("limit", "100")
	params["limit"] = limit

	offset := request.GetString("offset", "0")
	if offset != "" {
		params["offset"] = offset
	}

	endpoint := "https://api.sejm.gov.pl/eli/acts/search"
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve acts by publisher '%s': %v", publisher, err)), nil
	}

	var searchResult struct {
		Items []eli.Act `json:"items"`
		Count int       `json:"count"`
	}
	if err := json.Unmarshal(data, &searchResult); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse acts data: %v", err)), nil
	}

	// Create summary and analysis
	var summary []string
	var results []string
	var nextActions []string

	summary = append(summary, fmt.Sprintf("Retrieved %d acts from publisher '%s'", len(searchResult.Items), publisher))
	summary = append(summary, fmt.Sprintf("Showing results %s-%d", offset, parseInt(offset)+len(searchResult.Items)))

	// Analyze years and document types
	yearCount := make(map[int]int)
	typeCount := make(map[string]int)

	for _, act := range searchResult.Items {
		if act.Year != nil {
			yearCount[int(*act.Year)]++
		}
		if act.Type != nil {
			typeCount[*act.Type]++
		}
	}

	// Show sample acts and statistics
	results = append(results, fmt.Sprintf("Publisher '%s' Legal Output Analysis:", publisher))

	if len(yearCount) > 0 {
		results = append(results, "\nYearly Distribution (top 5 years):")
		years := make([]int, 0, len(yearCount))
		for year := range yearCount {
			years = append(years, year)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(years)))

		for i, year := range years {
			if i >= 5 {
				break
			}
			results = append(results, fmt.Sprintf("  %d: %d acts", year, yearCount[year]))
		}
	}

	// Show sample acts
	results = append(results, "\nSample Acts (first 10):")
	for i, act := range searchResult.Items {
		if i >= 10 {
			break
		}

		title := "No title"
		if act.Title != nil {
			title = *act.Title
		}

		year := "Unknown"
		if act.Year != nil {
			year = fmt.Sprintf("%d", *act.Year)
		}

		results = append(results, fmt.Sprintf("%d. %s (%s)", i+1, title, year))
	}

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("Get yearly breakdown: eli_get_acts_by_year with publisher='%s'", publisher))
	nextActions = append(nextActions, "Use eli_get_act_details for specific act information")
	if len(searchResult.Items) == parseInt(limit) {
		nextActions = append(nextActions, fmt.Sprintf("Continue browsing: use offset='%d' for more results", parseInt(offset)+parseInt(limit)))
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Acts by Publisher: %s", publisher),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Publisher-specific acts retrieved on %s.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetActsByYear(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("eli_get_acts_by_year called", slog.Any("arguments", request.Params.Arguments))

	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")

	if publisher == "" || year == "" {
		return mcp.NewToolResultError("Both 'publisher' and 'year' parameters are required. Get publisher codes from eli_get_publishers."), nil
	}

	params := make(map[string]string)
	limit := request.GetString("limit", "100")
	params["limit"] = limit

	offset := request.GetString("offset", "0")
	if offset != "" {
		params["offset"] = offset
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/eli/acts/%s/%s", publisher, year)
	data, err := s.makeAPIRequestWithHeaders(ctx, endpoint, params, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve acts for %s/%s: %v", publisher, year, err)), nil
	}

	var actsResponse eli.Acts
	if err := json.Unmarshal(data, &actsResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse acts data: %v", err)), nil
	}

	// Create comprehensive analysis
	var summary []string
	var results []string
	var nextActions []string

	acts := []eli.ActInfo{}
	if actsResponse.Items != nil {
		acts = *actsResponse.Items
	}

	summary = append(summary, fmt.Sprintf("Retrieved %d acts from %s in %s", len(acts), publisher, year))

	// Analyze document types
	typeCount := make(map[string]int)
	for _, act := range acts {
		if act.Type != nil {
			typeCount[*act.Type]++
		}
	}

	if len(typeCount) > 0 {
		summary = append(summary, fmt.Sprintf("Document types: %d different types found", len(typeCount)))
	}

	// Show document type breakdown
	results = append(results, fmt.Sprintf("Legislative Output for %s %s:", publisher, year))

	if len(typeCount) > 0 {
		results = append(results, "\nDocument Type Distribution:")
		types := make([]string, 0, len(typeCount))
		for docType := range typeCount {
			types = append(types, docType)
		}
		sort.Strings(types)

		for _, docType := range types {
			results = append(results, fmt.Sprintf("  %s: %d acts", docType, typeCount[docType]))
		}
	}

	// Show sample acts
	results = append(results, "\nSample Acts (first 10):")
	for i, act := range acts {
		if i >= 10 {
			break
		}

		title := "No title"
		if act.Title != nil {
			title = *act.Title
		}

		docType := "Unknown type"
		if act.Type != nil {
			docType = *act.Type
		}

		results = append(results, fmt.Sprintf("%d. %s (%s)", i+1, title, docType))
	}

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("Compare with other years: eli_get_acts_by_publisher with publisher='%s'", publisher))
	nextActions = append(nextActions, "Use eli_get_act_details for specific act information")
	if len(acts) == parseInt(limit) {
		nextActions = append(nextActions, fmt.Sprintf("Continue browsing: use offset='%d' for more results", parseInt(offset)+parseInt(limit)))
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Acts by Year: %s %s", publisher, year),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Yearly acts analysis retrieved on %s.", time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

// Helper function to parse string to int
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}
