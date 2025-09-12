package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/janisz/sejm-mcp/pkg/sejm"
	"github.com/mark3labs/mcp-go/mcp"
)

const sejmBaseURL = "https://api.sejm.gov.pl"

func (s *SejmServer) registerSejmTools() {
	s.registerProcessesTools()
	s.registerBilateralGroupsTools()
	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_terms",
		Description: "Retrieve list of all parliamentary terms with their duration, dates, and status information. Returns comprehensive information about each Sejm term including start/end dates, current status, number of sittings, and key statistics. Essential for understanding the structure of Polish parliamentary history, analyzing legislative periods, and contextualizing political developments over time.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
		},
	}, s.handleGetTerms)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_clubs",
		Description: "Retrieve comprehensive list of parliamentary clubs (political parties and groups) for a specific term. Returns detailed information about each club including full names, membership counts, formation dates, logos, and current status. Parliamentary clubs represent the main political groupings in the Sejm and are essential for understanding political dynamics, coalition structures, voting patterns, and party representation.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term has different club compositions due to elections and political changes. Current term 10 covers 2019-2023.",
				},
			},
		},
	}, s.handleGetClubs)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_club_details",
		Description: "Retrieve detailed information about a specific parliamentary club (political party/group). Returns comprehensive club data including full name, abbreviations, membership details, formation history, leadership structure, contact information, and current status. Essential for detailed political analysis, understanding party structures, researching specific political organizations, and analyzing club composition changes over time.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the club was active.",
				},
				"club_id": map[string]interface{}{
					"type":        "string",
					"description": "Club identifier. Get this from sejm_get_clubs results (the 'id' field).",
				},
			},
			Required: []string{"term", "club_id"},
		},
	}, s.handleGetClubDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_voting_details",
		Description: "Get detailed information about a specific parliamentary voting including vote counts, MP-by-MP voting records, voting title, topic, date, and outcome. When PDF format is available, automatically converts to searchable text with page location mapping. Essential for analyzing voting patterns, party discipline, individual MP behavior, and understanding specific legislative decisions in detail.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 voting activity.",
				},
				"sitting": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary sitting number (e.g., '1', '15', '30'). Get this from sejm_search_votings results.",
				},
				"voting_number": map[string]interface{}{
					"type":        "string",
					"description": "Specific voting number within the sitting (e.g., '1', '2', '5'). Get this from sejm_search_votings results.",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Response format: 'json' for structured data (default), 'text' for PDF converted to searchable text with page numbers, 'pdf' for raw PDF download.",
				},
			},
			Required: []string{"sitting", "voting_number"},
		},
	}, s.handleGetVotingDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_written_questions",
		Description: "Retrieve parliamentary written questions (zapytania) - formal written inquiries submitted by MPs to government ministers. Written questions are similar to interpellations but typically require shorter response times. Returns detailed information including question title, submitting MP(s), target ministry/minister, submission and response dates, current status, and government replies. Essential for monitoring government accountability, tracking ministerial responsiveness, analyzing MP oversight activity, and researching specific policy concerns.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 oversight activity. Each term reflects different political dynamics and government accountability patterns.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of written questions to return (default: 20). Use higher values (e.g., '50', '100') for comprehensive oversight analysis, but be aware of context limits.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through large datasets. For example, offset='50' with limit='50' returns results 51-100.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort written questions by specified field. Add minus sign for descending order (e.g., '-lastModified' for newest first, 'title' for alphabetical). Common fields: 'lastModified', 'title', 'receiptDate'.",
				},
				"from": map[string]interface{}{
					"type":        "string",
					"description": "Filter written questions from a MP with a specified ID. Get MP IDs from sejm_get_mps results.",
				},
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Filter written questions sent to a specified recipient (ministry or minister name).",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Filter written questions containing a specified string in the title.",
				},
				"since": map[string]interface{}{
					"type":        "string",
					"description": "Filter written questions starting from a specified date (YYYY-MM-DD format).",
				},
				"till": map[string]interface{}{
					"type":        "string",
					"description": "Filter written questions ending before a specified date (YYYY-MM-DD format).",
				},
				"delayed": map[string]interface{}{
					"type":        "string",
					"description": "Set to 'true' to display only cases where an answer is delayed beyond the statutory response time.",
				},
			},
		},
	}, s.handleGetWrittenQuestions)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_search_voting_content",
		Description: "Search for specific text within parliamentary voting documents and get precise page locations. Downloads voting PDFs, searches for specified terms, and returns detailed map showing exactly which pages contain each search term. Perfect for quickly locating specific MPs, voting topics, or legislative details within large voting documents without reading the entire text.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023.",
				},
				"sitting": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary sitting number. Must match exactly with the voting document.",
				},
				"voting_number": map[string]interface{}{
					"type":        "string",
					"description": "Specific voting number within the sitting.",
				},
				"search_terms": map[string]interface{}{
					"type":        "string",
					"description": "Search terms separated by commas. Can include MP names, party names, voting topics, or any text. Examples: 'Kowalski,PiS,za' or 'konstytucja,artykuł,przeciw'.",
				},
				"context_chars": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Number of characters to show around each match for context (default: 100, max: 500).",
				},
				"max_matches_per_term": map[string]interface{}{
					"type":        "string",
					"description": "Optional. Maximum number of matches to show per search term (default: 10, max: 50).",
				},
			},
			Required: []string{"sitting", "voting_number", "search_terms"},
		},
	}, s.handleSearchVotingContent)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_proceedings",
		Description: "Retrieve list of parliamentary proceedings (sessions) for a specific term, sorted by most recent first. Returns detailed information about each proceeding including dates, duration, topics discussed, and current status. Parliamentary proceedings represent the main sessions where MPs gather to debate, vote, and conduct official business. Essential for understanding current parliamentary activity, tracking legislative progress, and analyzing the timing of political decisions. Term 10 includes current 2025 parliamentary sessions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 proceedings.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of proceedings to return (default: 20). Use higher values for comprehensive analysis but be aware of context limits.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination. Since results are sorted by most recent first, offset='20' with limit='20' shows proceedings 21-40.",
				},
			},
		},
	}, s.handleGetProceedings)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_current_proceeding",
		Description: "Retrieve information about the current active parliamentary proceeding (session). Returns details about the proceeding currently in progress or most recently concluded, including proceeding number, date, status, topics being discussed, and timing information. Essential for real-time parliamentary monitoring, understanding current legislative activity, tracking live debates, and staying updated on immediate parliamentary business.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers current parliamentary activity.",
				},
			},
			Required: []string{"term"},
		},
	}, s.handleGetCurrentProceeding)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_prints",
		Description: "Retrieve parliamentary prints (legislative documents, bills, reports) for a specific term. Returns comprehensive information about each print including title, type, submitting MPs/institutions, submission date, current status in legislative process, and document details. Prints are the formal documents that contain proposed legislation, committee reports, government bills, and other official parliamentary documents. Critical for tracking legislative proposals, analyzing lawmaking process, and understanding the flow of political initiatives.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 legislative documents.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of prints to return (default: 30). Use higher values for comprehensive legislative analysis, but be aware of context limits.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through legislative documents.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort prints by specified field. Add minus sign for descending order (e.g., '-lastModified' for newest first, 'title' for alphabetical). Common fields: 'lastModified', 'title', 'number'.",
				},
			},
		},
	}, s.handleGetPrints)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_print_details",
		Description: "Retrieve detailed information about a specific parliamentary print (legislative document). Returns comprehensive information including print title, description, submitting institution/MPs, submission date, current status in legislative process, document type, related proceedings, and complete metadata. Essential for tracking specific legislation, analyzing legislative proposals, understanding document flow through parliament, and researching the history and details of particular bills or reports.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the print was submitted.",
				},
				"num": map[string]interface{}{
					"type":        "string",
					"description": "Print number. Get this from sejm_get_prints results (the 'number' field).",
				},
			},
			Required: []string{"term", "num"},
		},
	}, s.handleGetPrintDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_print_attachment",
		Description: "Download attachment files associated with parliamentary prints. Returns binary file content (PDFs, documents, images) that are attached to legislative documents and bills. Essential for accessing the full text of proposed legislation, supporting documentation, amendments, committee reports, legal analyses, and other materials that supplement the print metadata. Use this to get complete context and detailed content for print analysis.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the print was submitted.",
				},
				"num": map[string]interface{}{
					"type":        "string",
					"description": "Print number. Get this from sejm_get_prints results.",
				},
				"attach_name": map[string]interface{}{
					"type":        "string",
					"description": "Attachment file name. Get this from print details (attachments array).",
				},
			},
			Required: []string{"term", "num", "attach_name"},
		},
	}, s.handleGetPrintAttachment)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_mps",
		Description: "Retrieve comprehensive list of Members of Parliament (MPs) for a specific parliamentary term. Returns detailed information about all MPs including their personal details, political party affiliation, electoral district, contact information, and current activity status. This tool is essential for political analysis, research on parliamentary composition, and understanding the current makeup of the Polish Parliament. Use this to identify MPs by party, region, or activity status, or to get a complete overview of parliamentary representation.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term lasts 4 years. Term 10 is current (2019-2023). Term 9 was 2015-2019, Term 8 was 2011-2015, etc. If not specified, defaults to current term (10). Use '10' for most recent data.",
				},
			},
		},
	}, s.handleGetMPs)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_mp_details",
		Description: "Get comprehensive biographical and political information about a specific Member of Parliament. Returns detailed profile including full name variations (for Polish grammar cases), birth information, education level, profession, electoral district details, political party membership, voting statistics, contact information, and current mandate status. Essential for creating MP profiles, analyzing individual political careers, verifying MP credentials, or researching specific politicians. Use this after getting MP list to drill down into individual MPs of interest.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Defaults to current term (10) if not specified. Different terms may have different MPs due to elections or mandate changes.",
				},
				"mp_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique MP identification number within the specified term. Get this ID from sejm_get_mps tool first. Each MP has a unique numeric ID that identifies them within their term (e.g., '1', '2', '123').",
				},
			},
			Required: []string{"mp_id"},
		},
	}, s.handleGetMPDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committees",
		Description: "Retrieve complete list of parliamentary committees with their structure, membership, and operational details. Returns information about standing committees (permanent), extraordinary committees (special purpose), and investigative committees. Each committee entry includes its official name, code, appointed members with their roles (chairman, deputy chairman, secretary, regular member), scope of work, contact information, appointment dates, and any subcommittees. Critical for understanding parliamentary workflow, tracking which MPs work on which policy areas, analyzing committee composition by party, and identifying subject matter experts among MPs.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Committee structure can change between terms. Current term is 10. Use this to see how committee organization has evolved over different parliamentary periods.",
				},
			},
		},
	}, s.handleGetCommittees)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committee_details",
		Description: "Retrieve detailed information about a specific parliamentary committee. Returns comprehensive committee data including full name, description, scope of work, complete membership list with roles (chairman, deputy chairman, secretary, members), appointment dates, contact information, subcommittees, and current status. Essential for understanding committee structure, analyzing MP roles and responsibilities, researching policy expertise, and tracking committee composition changes.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the committee was active.",
				},
				"committee_code": map[string]interface{}{
					"type":        "string",
					"description": "Committee code (e.g., 'ENM', 'ASW', 'SUE'). Get this from sejm_get_committees results.",
				},
			},
			Required: []string{"term", "committee_code"},
		},
	}, s.handleGetCommitteeDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_search_votings",
		Description: "Search and analyze parliamentary voting records with detailed vote counts and outcomes. Returns comprehensive voting data including vote title, topic, description, voting type (electronic/traditional/on list), date and time, sitting information, vote tallies (yes/no/abstain/not participating), majority type required, and whether the vote passed. Essential for political analysis, tracking MP voting patterns, analyzing party discipline, studying legislative success rates, measuring parliamentary attendance, and understanding decision-making processes. Use this to research specific legislation votes, analyze voting trends, or track controversial decisions.\n\nIMPORTANT: You must provide EITHER 'sitting' OR 'title' parameter (not both, not neither). Use 'sitting' to get all votes from a specific parliamentary session, or 'title' to search across multiple sessions for votes matching keywords.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term has different voting records. Current term 10 covers 2019-2023 voting activity.",
				},
				"sitting": map[string]interface{}{
					"type":        "string",
					"description": "Specific parliamentary sitting number to get detailed votes from that sitting (e.g., '1', '2', '15', '25'). When provided, returns actual voting records with titles, vote counts, and results from that sitting. Recent sittings for term 10: 1-50+ are available. Use this when you want comprehensive voting data from a specific parliamentary session. MUTUALLY EXCLUSIVE with 'title' parameter.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Search for votes containing specific keywords in their titles or topics (e.g., 'budget', 'ustawa', 'projekt', 'konstytucja'). Searches across recent proceedings (last 20 sessions) for matching votes. Use this to find votes on specific topics or legislation across multiple sittings. MUTUALLY EXCLUSIVE with 'sitting' parameter.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of voting records to return (default: 20). Use higher values (e.g., '50', '100') for comprehensive analysis, lower values for quick overviews.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Number of results to skip for pagination (default: 0). Use with limit for browsing through large result sets. Example: offset='50' with limit='20' gets results 51-70.",
				},
				"date_from": map[string]interface{}{
					"type":        "string",
					"description": "Start date for voting search in YYYY-MM-DD format (e.g., '2023-01-01'). Only returns votes from this date onwards. Use with date_to for date range searches.",
				},
				"date_to": map[string]interface{}{
					"type":        "string",
					"description": "End date for voting search in YYYY-MM-DD format (e.g., '2023-12-31'). Only returns votes up to this date. Use with date_from for date range searches.",
				},
			},
		},
	}, s.handleSearchVotings)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_interpellations",
		Description: "Retrieve parliamentary interpellations - formal written questions submitted by MPs to government ministers requiring official responses. These are a key tool of parliamentary oversight and government accountability. Returns detailed information including question title, submitting MP(s), target ministry/minister, submission and response dates, current status, response delays, and government replies. Critical for monitoring government accountability, tracking ministerial responsiveness, analyzing MP oversight activity, identifying policy concerns, researching government performance, and studying democratic accountability mechanisms. Use this to investigate government responsiveness, track specific policy issues, or analyze MP engagement with executive oversight.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 oversight activity. Each term reflects different political dynamics and government accountability patterns.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of interpellations to return (default: 20). Use higher values (e.g., '50', '100') for comprehensive oversight analysis, but be aware of context limits. Large datasets useful for trend analysis and accountability studies.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through large datasets. For example, offset='50' with limit='50' returns results 51-100.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort interpellations by specified field. Add minus sign for descending order (e.g., '-lastModified' for newest first, 'title' for alphabetical). Common fields: 'lastModified', 'title', 'receiptDate'.",
				},
			},
		},
	}, s.handleGetInterpellations)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_interpellation_body",
		Description: "Retrieve the full HTML body content of a specific parliamentary interpellation. Returns the complete text of the interpellation question as submitted by MPs to government ministers. Essential for analyzing the detailed content, specific questions asked, legal references cited, and policy concerns raised. Use this after finding interpellations with sejm_get_interpellations to get the full question text for detailed analysis, research, or transparency reporting.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the interpellation was submitted.",
				},
				"num": map[string]interface{}{
					"type":        "string",
					"description": "Interpellation number. Get this from sejm_get_interpellations results (the 'num' field).",
				},
			},
			Required: []string{"term", "num"},
		},
	}, s.handleGetInterpellationBody)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_interpellation_reply_body",
		Description: "Retrieve the full HTML body content of a government reply to a parliamentary interpellation. Returns the complete ministerial response including policy explanations, statistical data, legal interpretations, and action plans. Critical for analyzing government accountability, policy responses, ministerial performance, and the quality of democratic oversight. Use this to examine how thoroughly government addresses MP concerns and parliamentary questions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the interpellation was submitted.",
				},
				"num": map[string]interface{}{
					"type":        "string",
					"description": "Interpellation number. Get this from sejm_get_interpellations results.",
				},
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Reply key/identifier. Get this from the interpellation details (replies array in sejm_get_interpellations results).",
				},
			},
			Required: []string{"term", "num", "key"},
		},
	}, s.handleGetInterpellationReplyBody)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_interpellation_attachment",
		Description: "Download attachment files associated with parliamentary interpellations. Returns binary file content (PDFs, documents, images) that MPs include with their interpellations or that ministries attach to their replies. Essential for accessing supporting documentation, legal references, statistical data, charts, reports, and evidence that supplement the interpellation text. Use this to get complete context and supporting materials for interpellation analysis.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Must match the term where the interpellation was submitted.",
				},
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Attachment key/identifier. Get this from interpellation details (attachments array).",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "Attachment file name. Get this from interpellation details (attachments array).",
				},
			},
			Required: []string{"term", "key", "file_name"},
		},
	}, s.handleGetInterpellationAttachment)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_transcripts",
		Description: "Retrieve parliamentary proceeding transcripts - complete stenographic records of parliamentary debates, speeches, and discussions. Returns detailed transcript information including individual MP statements, speech timestamps, debate topics, speaker identification, and full text content. For large PDF transcripts, use pagination parameters (page, pages_per_chunk) to manage response size and avoid context overflow. Essential for analyzing parliamentary debates, tracking MP positions on issues, studying political discourse, researching specific policy discussions, and understanding the legislative decision-making process.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 parliamentary debates.",
				},
				"proceeding_id": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary proceeding/sitting number (e.g., '1', '15', '30'). Get this from sejm_get_proceedings results.",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Proceeding date in YYYY-MM-DD format (e.g., '2023-11-13'). Get this from sejm_get_proceedings results.",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Response format: 'list' for statement list (default), 'pdf' for complete transcript as PDF, 'text' for PDF converted to searchable text.",
				},
				"page": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Starting page number (1-based). Use with pages_per_chunk to control output size. Default: 1.",
				},
				"pages_per_chunk": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Number of pages to include per response (1-10). Helps manage large transcript responses. Default: 5.",
				},
				"show_page_info": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Set to 'true' to show page count and navigation info instead of content. Useful for understanding document structure.",
				},
			},
			Required: []string{"proceeding_id", "date"},
		},
	}, s.handleGetTranscripts)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_statement",
		Description: "Retrieve individual MP statement from parliamentary transcript - complete text of a specific speech or intervention during parliamentary proceedings. Returns detailed statement content including speaker information, timestamp, full text, context within the debate, and related discussion. Essential for analyzing specific MP positions, studying individual political statements, researching particular policy arguments, and understanding detailed parliamentary discourse. Use this to get the complete text of specific speeches or interventions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 statements.",
				},
				"proceeding_id": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary proceeding/sitting number. Get this from transcript list results.",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Proceeding date in YYYY-MM-DD format. Get this from transcript list results.",
				},
				"statement_num": map[string]interface{}{
					"type":        "string",
					"description": "Statement number within the proceeding (e.g., '1', '5', '23'). Get this from sejm_get_transcripts results.",
				},
				"chunk_size": map[string]interface{}{
					"type":        "string",
					"description": "For large HTML responses: Number of characters per chunk (1000-10000). Default: 5000. Helps manage large statement responses.",
				},
				"chunk_number": map[string]interface{}{
					"type":        "string",
					"description": "For large HTML responses: Which chunk to return (1-based). Default: 1. Use with chunk_size to paginate through large statements.",
				},
				"show_chunk_info": map[string]interface{}{
					"type":        "string",
					"description": "Set to 'true' to show total chunks and navigation info instead of content. Useful for understanding statement structure.",
				},
			},
			Required: []string{"proceeding_id", "date", "statement_num"},
		},
	}, s.handleGetStatement)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committee_sittings_by_date",
		Description: "Retrieve list of committee meetings scheduled for a specific date across all committees. Returns comprehensive information about parliamentary committee activities including meeting times, rooms, agendas, and committee codes. Essential for tracking daily parliamentary committee work, scheduling analysis, and understanding committee activity patterns. Use this to see all committee meetings happening on a particular day.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 committee activities.",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Meeting date in YYYY-MM-DD format (e.g., '2023-11-20'). Returns all committee meetings scheduled for this date.",
				},
				"canceled": map[string]interface{}{
					"type":        "string",
					"description": "Set to 'true' to include canceled meetings in results. Default: false (only active meetings).",
				},
			},
			Required: []string{"date"},
		},
	}, s.handleGetCommitteeSittingsByDate)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committee_sittings",
		Description: "Retrieve list of meetings for a specific parliamentary committee. Returns detailed information about committee meeting history including dates, agenda items, participants, and meeting outcomes. Essential for tracking specific committee work, analyzing committee productivity, and understanding legislative committee processes. Use this to explore the complete meeting history of a particular committee.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 committee meetings.",
				},
				"committee_code": map[string]interface{}{
					"type":        "string",
					"description": "Committee code (e.g., 'ENM', 'ASW', 'SUE'). Get this from sejm_get_committees results. Each committee has a unique code identifier.",
				},
				"canceled": map[string]interface{}{
					"type":        "string",
					"description": "Set to 'true' to include canceled meetings in results. Default: false (only completed meetings).",
				},
			},
			Required: []string{"committee_code"},
		},
	}, s.handleGetCommitteeSittings)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committee_sitting_details",
		Description: "Get detailed information about a specific committee meeting including agenda, participants, decisions, and meeting metadata. Returns comprehensive sitting details with timestamps, attendees, topics discussed, and outcomes. Essential for analyzing specific committee decisions, understanding committee workflow, and researching detailed committee proceedings.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 committee meetings.",
				},
				"committee_code": map[string]interface{}{
					"type":        "string",
					"description": "Committee code (e.g., 'ENM', 'ASW'). Get this from committee listings or sitting results.",
				},
				"sitting_number": map[string]interface{}{
					"type":        "string",
					"description": "Meeting number within the committee (e.g., '1', '5', '15'). Get this from committee sitting lists.",
				},
			},
			Required: []string{"committee_code", "sitting_number"},
		},
	}, s.handleGetCommitteeSittingDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_committee_transcript",
		Description: "Retrieve committee meeting transcripts in HTML or PDF format with pagination support for large documents. Returns complete stenographic records of committee discussions, member statements, expert testimonies, and voting records. For large transcripts, use pagination parameters to manage response size and avoid context overflow. Essential for detailed analysis of committee work, policy development research, and understanding legislative decision-making processes.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 committee transcripts.",
				},
				"committee_code": map[string]interface{}{
					"type":        "string",
					"description": "Committee code (e.g., 'ENM', 'ASW'). Get this from committee listings.",
				},
				"sitting_number": map[string]interface{}{
					"type":        "string",
					"description": "Meeting number within the committee. Get this from committee sitting lists.",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Response format: 'html' for HTML transcript (default), 'pdf' for PDF download info, 'text' for PDF converted to searchable text with pagination.",
				},
				"page": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Starting page number (1-based). Use with pages_per_chunk to control output size.",
				},
				"pages_per_chunk": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Number of pages to include per response (1-10). Default: 5.",
				},
				"show_page_info": map[string]interface{}{
					"type":        "string",
					"description": "For 'text' format: Set to 'true' to show page count and navigation info instead of content.",
				},
				"chunk_size": map[string]interface{}{
					"type":        "string",
					"description": "For 'html' format: Characters per chunk (1000-10000). Default: 5000.",
				},
				"chunk_number": map[string]interface{}{
					"type":        "string",
					"description": "For 'html' format: Which chunk to return (1-based). Default: 1.",
				},
				"show_chunk_info": map[string]interface{}{
					"type":        "string",
					"description": "For 'html' format: Set to 'true' to show document structure info instead of content.",
				},
			},
			Required: []string{"committee_code", "sitting_number"},
		},
	}, s.handleGetCommitteeTranscript)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_mp_photo",
		Description: "Get MP (Member of Parliament) official photo in full size. Returns the MP's parliamentary portrait photo used in official documents and parliamentary materials. These photos are standardized parliamentary portraits that provide visual identification of MPs for democratic transparency and public accountability. Useful for creating MP profiles, media materials, parliamentary documentation, or citizen information resources.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term has different MPs due to elections. Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
				"mp_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique MP identification number within the specified term. Get this ID from sejm_get_mps tool first. Each MP has a unique numeric ID that identifies them within their term (e.g., '1', '2', '123').",
				},
				"size": map[string]interface{}{
					"type":        "string",
					"description": "Photo size: 'full' for standard parliamentary portrait (default), 'mini' for smaller thumbnail version suitable for lists or compact displays.",
				},
			},
			Required: []string{"mp_id"},
		},
	}, s.handleGetMPPhoto)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_mp_voting_stats",
		Description: "Get comprehensive voting statistics for a specific Member of Parliament including attendance rates, participation patterns, and voting behavior analysis. Returns detailed statistical data about the MP's parliamentary activity including sitting attendance, voting participation rates, excuse patterns, and overall engagement metrics. Essential for analyzing MP performance, democratic accountability research, parliamentary oversight, citizen engagement, and transparency reporting. Use this to assess individual MP accountability, compare MP activity levels, or analyze parliamentary attendance patterns.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term has different voting patterns and MPs. Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
				"mp_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique MP identification number within the specified term. Get this ID from sejm_get_mps tool first. Each MP has a unique numeric ID that identifies them within their term (e.g., '1', '2', '123').",
				},
			},
			Required: []string{"mp_id"},
		},
	}, s.handleGetMPVotingStats)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_mp_voting_details",
		Description: "Get detailed voting records for a specific Member of Parliament during a particular parliamentary sitting. Returns comprehensive vote-by-vote information including specific voting choices (yes/no/abstain/absent), vote titles, topics, timestamps, and voting context. Essential for analyzing individual MP voting behavior, tracking specific legislative positions, researching MP consistency on issues, understanding party discipline, and conducting detailed political accountability analysis. Use this to examine how an MP voted on specific legislation or during important parliamentary sessions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
				"mp_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique MP identification number within the specified term. Get this ID from sejm_get_mps tool first.",
				},
				"sitting": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary sitting number (e.g., '1', '15', '30'). Get this from sejm_search_votings or sejm_get_proceedings results.",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Sitting date in YYYY-MM-DD format (e.g., '2023-12-13'). Must match the actual date of the specified sitting.",
				},
			},
			Required: []string{"mp_id", "sitting", "date"},
		},
	}, s.handleGetMPVotingDetails)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_videos",
		Description: "Retrieve parliamentary video transmissions and live streams with comprehensive filtering options. Returns detailed information about video broadcasts including live parliamentary sessions, committee meetings, special events, and archived proceedings. Each video entry includes streaming URLs, player links, transmission metadata, schedules, and technical details. Essential for accessing live parliamentary coverage, following specific committee work, researching historical proceedings, media monitoring, and democratic transparency. Use this to find current live streams, search for specific meeting recordings, or track parliamentary video activity.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Each term has different video coverage and technology. Current term 10 covers 2019-2023 with modern streaming infrastructure.",
				},
				"committee": map[string]interface{}{
					"type":        "string",
					"description": "Committee code to filter videos for specific committee meetings (e.g., 'SUE', 'ENM', 'ASW'). Get committee codes from sejm_get_committees.",
				},
				"since": map[string]interface{}{
					"type":        "string",
					"description": "Start date filter in YYYY-MM-DD format (e.g., '2022-01-15'). Returns videos from this date onwards.",
				},
				"till": map[string]interface{}{
					"type":        "string",
					"description": "End date filter in YYYY-MM-DD format (e.g., '2022-01-15'). Returns videos up to this date.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Search term to filter videos by title content (e.g., 'kodeksu pracy', 'budżet'). Case-insensitive substring matching.",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Transmission type filter (e.g., 'komisja' for committee meetings, 'posiedzenie' for plenary sessions). Common types include committee meetings, plenary sessions, special events.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of videos to return (default: 20, max: 500). Use higher values for comprehensive searches, but be aware of context limits.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position in results for pagination (default: 0). Use with limit for browsing large result sets.",
				},
			},
		},
	}, s.handleGetVideos)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_videos_today",
		Description: "Get today's parliamentary video transmissions and live streams including current live sessions, scheduled broadcasts, and ongoing parliamentary activities. Returns real-time information about active video streams, upcoming transmissions, and current parliamentary events with streaming links and schedules. Essential for following current parliamentary activity, accessing live coverage, monitoring ongoing debates, and staying informed about real-time democratic processes. Perfect for media monitoring, civic engagement, and immediate parliamentary coverage.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
			},
		},
	}, s.handleGetVideosToday)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_videos_by_date",
		Description: "Retrieve parliamentary video transmissions for a specific date including all sessions, committee meetings, and special events that occurred on that day. Returns comprehensive video coverage information with streaming URLs, archived recordings, meeting metadata, and transmission details. Essential for researching historical parliamentary activity, accessing archived proceedings, studying specific legislative sessions, and understanding parliamentary events on particular dates. Use this to find recordings of important votes, committee meetings, or special parliamentary events.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format (e.g., '2023-12-13'). Returns all video transmissions that occurred on this specific date.",
				},
			},
			Required: []string{"date"},
		},
	}, s.handleGetVideosByDate)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_video_details",
		Description: "Get detailed metadata and streaming information for a specific video transmission including direct streaming URLs, player links, technical specifications, transmission schedule, and comprehensive event details. Returns complete video transmission data with multiple camera angles, sign language streams, player embed codes, and full technical metadata. Essential for accessing specific video content, embedding streams, technical integration, detailed media analysis, and comprehensive parliamentary video research.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023. Defaults to current term (10) if not specified.",
				},
				"unid": map[string]interface{}{
					"type":        "string",
					"description": "Unique video transmission identifier (32-character alphanumeric string, e.g., '2A8A86E819C2C270C1258ACB0047A157'). Get this from video listing results.",
				},
			},
			Required: []string{"unid"},
		},
	}, s.handleGetVideoDetails)
}

func (s *SejmServer) handleGetMPs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10, where 10 is the current term (2019-2023), 9 was 2015-2019, etc.", err)), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/MP", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve MPs from Polish Parliament API: %v. Please try again or check if the term number is valid.", err)), nil
	}

	var mps []sejm.MP
	if err := json.Unmarshal(data, &mps); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse MP data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Create a summary instead of returning full data to avoid context overflow
	activeCount := 0
	partyStats := make(map[string]int)
	districtStats := make(map[string]int)

	// Create summary list with essential info only
	type MPSummary struct {
		ID           *int32  `json:"id"`
		Name         string  `json:"name"`
		Party        *string `json:"party,omitempty"`
		District     *int32  `json:"district,omitempty"`
		DistrictName *string `json:"districtName,omitempty"`
		Active       *bool   `json:"active,omitempty"`
	}

	var mpSummaries []MPSummary

	for _, mp := range mps {
		if mp.Active != nil && *mp.Active {
			activeCount++
		}

		// Count party affiliations
		if mp.Club != nil {
			partyStats[*mp.Club]++
		}

		// Count districts
		if mp.DistrictName != nil {
			districtStats[*mp.DistrictName]++
		}

		// Create summary entry
		name := getFullName(mp)
		mpSummaries = append(mpSummaries, MPSummary{
			ID:           mp.Id,
			Name:         name,
			Party:        mp.Club,
			District:     mp.DistrictNum,
			DistrictName: mp.DistrictName,
			Active:       mp.Active,
		})
	}

	summary := fmt.Sprintf("Parliamentary term %d MP overview:\n", term)
	summary += fmt.Sprintf("- Total MPs: %d (%d active, %d inactive)\n", len(mps), activeCount, len(mps)-activeCount)
	summary += "- Use sejm_get_mp_details with specific mp_id to get full details about any MP\n\n"

	// Add party breakdown
	summary += "Party composition:\n"
	for party, count := range partyStats {
		summary += fmt.Sprintf("- %s: %d MPs\n", party, count)
	}

	// Show first 20 MPs as examples
	summary += "\nFirst 20 MPs (use mp_id with sejm_get_mp_details for full info):\n"
	for i, mp := range mpSummaries {
		if i >= 20 {
			break
		}
		activeStatus := "inactive"
		if mp.Active != nil && *mp.Active {
			activeStatus = "active"
		}
		party := "No party"
		if mp.Party != nil {
			party = *mp.Party
		}
		summary += fmt.Sprintf("- ID %v: %s (%s) - %s\n", *mp.ID, mp.Name, party, activeStatus)
	}

	if len(mpSummaries) > 20 {
		summary += fmt.Sprintf("\n... and %d more MPs. Use sejm_get_mp_details with specific IDs for details.\n", len(mpSummaries)-20)
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetMPDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	mpID := request.GetString("mp_id", "")
	if mpID == "" {
		return mcp.NewToolResultError("MP ID is required. Please provide the mp_id parameter with a valid MP identification number. You can get MP IDs from the sejm_get_mps tool."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/MP/%s", sejmBaseURL, term, mpID)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve MP details from Polish Parliament API: %v. Please verify the MP ID (%s) exists in term %d. You can get valid MP IDs using sejm_get_mps.", err, mpID, term)), nil
	}

	var mp sejm.MP
	if err := json.Unmarshal(data, &mp); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse MP data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Build comprehensive description
	description := fmt.Sprintf("Detailed profile for MP %s (ID: %s) from parliamentary term %d:",
		getFullName(mp), mpID, term)

	if mp.Club != nil {
		description += fmt.Sprintf("\n- Political Party/Club: %s", *mp.Club)
	}
	if mp.DistrictName != nil {
		description += fmt.Sprintf("\n- Electoral District: %s", *mp.DistrictName)
	}
	if mp.Active != nil {
		status := "Active"
		if !*mp.Active {
			status = "Inactive"
		}
		description += fmt.Sprintf("\n- Current Status: %s", status)
	}
	if mp.Email != nil {
		description += fmt.Sprintf("\n- Contact: %s", *mp.Email)
	}

	result, _ := json.MarshalIndent(mp, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%s\n\nComplete MP data:\n%s", description, string(result))), nil
}

func getFullName(mp sejm.MP) string {
	if mp.FirstLastName != nil {
		return *mp.FirstLastName
	}
	if mp.FirstName != nil && mp.LastName != nil {
		return fmt.Sprintf("%s %s", *mp.FirstName, *mp.LastName)
	}
	return "Unknown"
}

func (s *SejmServer) handleGetCommittees(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/committees", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve committees from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var committees []sejm.Committee
	if err := json.Unmarshal(data, &committees); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse committee data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Analyze committee structure
	standingCount := 0
	extraordinaryCount := 0
	investigativeCount := 0

	for _, committee := range committees {
		if committee.Type != nil {
			switch *committee.Type {
			case "STANDING":
				standingCount++
			case "EXTRAORDINARY":
				extraordinaryCount++
			case "INVESTIGATIVE":
				investigativeCount++
			}
		}
	}

	// Create committee summary to avoid context overflow
	type CommitteeSummary struct {
		Code        *string            `json:"code,omitempty"`
		Name        *string            `json:"name,omitempty"`
		Type        *sejm.ComitteeType `json:"type,omitempty"`
		MemberCount int                `json:"memberCount"`
	}

	var committeeSummaries []CommitteeSummary
	for _, committee := range committees {
		memberCount := 0
		if committee.Members != nil {
			memberCount = len(*committee.Members)
		}

		committeeSummaries = append(committeeSummaries, CommitteeSummary{
			Code:        committee.Code,
			Name:        committee.Name,
			Type:        committee.Type,
			MemberCount: memberCount,
		})
	}

	summary := fmt.Sprintf("Parliamentary committee structure for term %d:\n", term)
	summary += fmt.Sprintf("- %d Standing Committees (permanent)\n", standingCount)
	summary += fmt.Sprintf("- %d Extraordinary Committees (special purpose)\n", extraordinaryCount)
	summary += fmt.Sprintf("- %d Investigative Committees\n", investigativeCount)
	summary += fmt.Sprintf("- Total: %d committees\n\n", len(committees))

	summary += "Committee list (code, name, type, member count):\n"
	for _, comm := range committeeSummaries {
		code := "N/A"
		if comm.Code != nil {
			code = *comm.Code
		}
		name := "Unknown"
		if comm.Name != nil {
			name = *comm.Name
		}
		commType := "Unknown"
		if comm.Type != nil {
			commType = string(*comm.Type)
		}
		summary += fmt.Sprintf("- %s: %s (%s, %d members)\n", code, name, commType, comm.MemberCount)
	}

	summary += "\nNote: This is a summary view. For detailed committee information including full membership, leadership roles, and contact details, use specific committee analysis tools or contact the API directly."

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleSearchVotings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	sitting := request.GetString("sitting", "")
	title := request.GetString("title", "")
	limit := request.GetString("limit", "20")

	var endpoint string
	var params map[string]string

	// Validate parameter requirements - exactly one of sitting or title must be provided
	if sitting != "" && title != "" {
		return mcp.NewToolResultError("Please provide EITHER 'sitting' OR 'title' parameter, not both. Use 'sitting' to get all votes from a specific parliamentary session (e.g., sitting='15'), or 'title' to search for votes matching keywords across multiple sessions (e.g., title='budget')."), nil
	}

	if sitting == "" && title == "" {
		return mcp.NewToolResultError("You must provide either 'sitting' or 'title' parameter:\n\n• Use 'sitting' parameter (e.g., '15', '25', '30') to get all voting records from a specific parliamentary session with detailed vote counts and titles\n• Use 'title' parameter (e.g., 'budget', 'ustawa', 'konstytucja') to search for votes containing specific keywords across recent sessions\n\nExamples:\n- sejm_search_votings with sitting='15' and term='10' (gets all votes from sitting 15)\n- sejm_search_votings with title='budget' and term='10' (finds budget-related votes)\n\nFor term 10, sitting numbers typically range from 1 to 50+. Try sitting='1' for early session votes or sitting='30' for more recent votes."), nil
	}

	// Choose the correct endpoint based on parameters
	if sitting != "" {
		// Get detailed votes from a specific sitting
		endpoint = fmt.Sprintf("%s/sejm/term%d/votings/%s", sejmBaseURL, term, sitting)
		params = nil
	} else {
		// Search for votes by title - implement client-side search
		// since the API search endpoint appears to be non-functional
		return s.searchVotingsByTitle(ctx, term, title, limit)
	}

	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		if sitting != "" {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve voting records from sitting %s in term %d: %v. Please verify the sitting number exists. For term 10, valid sitting numbers typically range from 1 to 50+. Try sitting='1' for early sessions, sitting='15' for mid-term sessions, or sitting='30' for recent sessions.", sitting, term, err)), nil
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve voting records for title search '%s' in term %d: %v. Please verify your search terms or try different keywords.", title, term, err)), nil
		}
	}

	var votings []sejm.Voting
	if err := json.Unmarshal(data, &votings); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse voting data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Limit results to avoid context overflow
	limitInt := 20
	if limit != "" {
		if parsedLimit, err := fmt.Sscanf(limit, "%d", &limitInt); parsedLimit != 1 || err != nil || limitInt <= 0 {
			limitInt = 20 // fallback to default
		}
	}
	if len(votings) > limitInt {
		votings = votings[:limitInt]
	}

	// Analyze voting patterns
	passedCount := 0
	electronicCount := 0
	traditionalCount := 0
	totalYes := 0
	totalNo := 0
	totalAbstain := 0

	for _, voting := range votings {
		if voting.Yes != nil && voting.No != nil {
			if *voting.Yes > *voting.No {
				passedCount++
			}
			totalYes += int(*voting.Yes)
			totalNo += int(*voting.No)
		}
		if voting.Abstain != nil {
			totalAbstain += int(*voting.Abstain)
		}
		if voting.Kind != nil {
			switch *voting.Kind {
			case "ELECTRONIC":
				electronicCount++
			case "TRADITIONAL":
				traditionalCount++
			}
		}
	}

	searchSummary := fmt.Sprintf("Voting records for parliamentary term %d", term)
	if sitting != "" {
		searchSummary += fmt.Sprintf(" (sitting %s)", sitting)
	}
	if title != "" {
		searchSummary += fmt.Sprintf(" (search: '%s')", title)
	}
	searchSummary += fmt.Sprintf(":\n- Found %d voting records (showing %d)", len(votings), len(votings))
	searchSummary += fmt.Sprintf("\n- %d votes passed, %d failed", passedCount, len(votings)-passedCount)
	searchSummary += fmt.Sprintf("\n- %d electronic votes, %d traditional votes", electronicCount, traditionalCount)
	searchSummary += fmt.Sprintf("\n- Total votes cast: %d Yes, %d No, %d Abstain\n\n", totalYes, totalNo, totalAbstain)

	// Show detailed voting results
	searchSummary += "Voting records (title, date, result, votes):\n"
	for i, voting := range votings {
		if i >= 15 { // Show first 15 to save space but provide meaningful data
			break
		}

		title := "No title"
		if voting.Title != nil {
			title = *voting.Title
		}

		date := "No date"
		if voting.Date != nil {
			date = voting.Date.Format("2006-01-02 15:04")
		}

		result := "Unknown"
		voteDetails := ""
		if voting.Yes != nil && voting.No != nil {
			if *voting.Yes > *voting.No {
				result = "PASSED"
			} else {
				result = "FAILED"
			}
			voteDetails = fmt.Sprintf("(%d Yes, %d No", *voting.Yes, *voting.No)
			if voting.Abstain != nil {
				voteDetails += fmt.Sprintf(", %d Abstain", *voting.Abstain)
			}
			voteDetails += ")"
		}

		searchSummary += fmt.Sprintf("- %s\n  %s - %s %s\n\n", title, date, result, voteDetails)
	}

	if len(votings) > 15 {
		searchSummary += fmt.Sprintf("... and %d more voting records. Use title search for more targeted results.", len(votings)-15)
	}

	return mcp.NewToolResultText(searchSummary), nil
}

func (s *SejmServer) handleGetInterpellations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	limit := request.GetString("limit", "20") // Reduced default to avoid context overflow
	params["limit"] = limit

	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}
	if sortBy := request.GetString("sort_by", ""); sortBy != "" {
		params["sort_by"] = sortBy
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/interpellations", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve interpellations from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var interpellations []sejm.Interpellation
	if err := json.Unmarshal(data, &interpellations); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse interpellation data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Analyze accountability patterns
	answeredCount := 0
	delayedCount := 0
	totalDelayDays := 0
	maxDelay := 0

	for _, interp := range interpellations {
		if interp.Replies != nil && len(*interp.Replies) > 0 {
			answeredCount++
		}
		if interp.AnswerDelayedDays != nil {
			delayDays := int(*interp.AnswerDelayedDays)
			if delayDays > 0 {
				delayedCount++
				totalDelayDays += delayDays
				if delayDays > maxDelay {
					maxDelay = delayDays
				}
			}
		}
	}

	avgDelay := 0
	if delayedCount > 0 {
		avgDelay = totalDelayDays / delayedCount
	}

	accountabilitySummary := fmt.Sprintf("Parliamentary oversight analysis for term %d:", term)
	accountabilitySummary += fmt.Sprintf("\n- %d interpellations found (limit: %s)", len(interpellations), limit)
	accountabilitySummary += fmt.Sprintf("\n- %d have received government responses (%.1f%%)", answeredCount, float64(answeredCount)*100/float64(len(interpellations)))
	accountabilitySummary += fmt.Sprintf("\n- %d responses were delayed", delayedCount)
	if delayedCount > 0 {
		accountabilitySummary += fmt.Sprintf("\n- Average delay: %d days, Maximum delay: %d days\n\n", avgDelay, maxDelay)
	} else {
		accountabilitySummary += "\n\n"
	}

	// Show interpellation summaries instead of full data
	accountabilitySummary += "Recent interpellations (title, submitter, status):\n"
	for i, interp := range interpellations {
		if i >= 10 { // Show only first 10 to save space
			break
		}
		title := "No title"
		if interp.Title != nil {
			title = *interp.Title
		}

		submitter := "Unknown"
		if interp.From != nil && len(*interp.From) > 0 {
			submitter = fmt.Sprintf("MP ID: %s", (*interp.From)[0])
		}

		status := "No response"
		if interp.Replies != nil && len(*interp.Replies) > 0 {
			status = "Answered"
		}

		accountabilitySummary += fmt.Sprintf("- %s (by %s) - %s\n", title, submitter, status)
	}

	if len(interpellations) > 10 {
		accountabilitySummary += fmt.Sprintf("\n... and %d more interpellations. Use a smaller limit for more targeted results.", len(interpellations)-10)
	}

	return mcp.NewToolResultText(accountabilitySummary), nil
}

func (s *SejmServer) searchVotingsByTitle(ctx context.Context, term int, titleSearch string, limitStr string) (*mcp.CallToolResult, error) {
	// First, get all voting sessions
	votingSessionsEndpoint := fmt.Sprintf("%s/sejm/term%d/votings", sejmBaseURL, term)
	sessionsData, err := s.makeAPIRequest(ctx, votingSessionsEndpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve voting sessions from Polish Parliament API: %v", err)), nil
	}

	var sessions []struct {
		Date       string `json:"date"`
		Proceeding int    `json:"proceeding"`
		VotingsNum int    `json:"votingsNum"`
	}
	if err := json.Unmarshal(sessionsData, &sessions); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse voting sessions data: %v", err)), nil
	}

	// Search through recent proceedings (limit to avoid excessive API calls)
	var allMatchingVotings []sejm.Voting
	searchedProceedings := 0
	maxProceedingsToSearch := 20 // Limit to recent proceedings to avoid timeouts

	for i := len(sessions) - 1; i >= 0 && searchedProceedings < maxProceedingsToSearch; i-- {
		session := sessions[i]
		if session.VotingsNum == 0 {
			continue
		}

		// Get detailed votings for this proceeding
		proceedingEndpoint := fmt.Sprintf("%s/sejm/term%d/votings/%d", sejmBaseURL, term, session.Proceeding)
		proceedingData, err := s.makeAPIRequest(ctx, proceedingEndpoint, nil)
		if err != nil {
			continue // Skip failed requests to avoid breaking the search
		}

		var votings []sejm.Voting
		if err := json.Unmarshal(proceedingData, &votings); err != nil {
			continue // Skip parsing errors
		}

		// Search for title matches (case-insensitive)
		titleLower := strings.ToLower(titleSearch)
		for _, voting := range votings {
			if voting.Title != nil && strings.Contains(strings.ToLower(*voting.Title), titleLower) {
				allMatchingVotings = append(allMatchingVotings, voting)
			}
			if voting.Topic != nil && strings.Contains(strings.ToLower(*voting.Topic), titleLower) {
				allMatchingVotings = append(allMatchingVotings, voting)
			}
		}

		searchedProceedings++
	}

	// Apply limit
	limitInt := 20
	if limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limitInt); parsedLimit != 1 || err != nil || limitInt <= 0 {
			limitInt = 20 // fallback to default
		}
	}
	if len(allMatchingVotings) > limitInt {
		allMatchingVotings = allMatchingVotings[:limitInt]
	}

	// Analyze voting patterns
	passedCount := 0
	electronicCount := 0
	traditionalCount := 0
	totalYes := 0
	totalNo := 0
	totalAbstain := 0

	for _, voting := range allMatchingVotings {
		if voting.Yes != nil && voting.No != nil {
			if *voting.Yes > *voting.No {
				passedCount++
			}
			totalYes += int(*voting.Yes)
			totalNo += int(*voting.No)
		}
		if voting.Abstain != nil {
			totalAbstain += int(*voting.Abstain)
		}
		if voting.Kind != nil {
			switch *voting.Kind {
			case "ELECTRONIC":
				electronicCount++
			case "TRADITIONAL":
				traditionalCount++
			}
		}
	}

	searchSummary := fmt.Sprintf("Voting search results for term %d (search: '%s'):", term, titleSearch)
	searchSummary += fmt.Sprintf("\n- Searched %d recent proceedings", searchedProceedings)
	searchSummary += fmt.Sprintf("\n- Found %d matching voting records (showing %d)", len(allMatchingVotings), len(allMatchingVotings))
	if len(allMatchingVotings) > 0 {
		searchSummary += fmt.Sprintf("\n- %d votes passed, %d failed", passedCount, len(allMatchingVotings)-passedCount)
		searchSummary += fmt.Sprintf("\n- %d electronic votes, %d traditional votes", electronicCount, traditionalCount)
		searchSummary += fmt.Sprintf("\n- Total votes cast: %d Yes, %d No, %d Abstain\n\n", totalYes, totalNo, totalAbstain)
	} else {
		searchSummary += "\n\nNo matching votes found. Try different keywords or broader search terms.\n\n"
	}

	// Show detailed voting results
	if len(allMatchingVotings) > 0 {
		searchSummary += "Matching voting records (title, date, result, votes):\n"
		for i, voting := range allMatchingVotings {
			if i >= 15 { // Show first 15 to save space
				break
			}

			title := "No title"
			if voting.Title != nil {
				title = *voting.Title
			}

			date := "No date"
			if voting.Date != nil {
				date = voting.Date.Format("2006-01-02 15:04")
			}

			result := "Unknown"
			voteDetails := ""
			if voting.Yes != nil && voting.No != nil {
				if *voting.Yes > *voting.No {
					result = "PASSED"
				} else {
					result = "FAILED"
				}
				voteDetails = fmt.Sprintf("(%d Yes, %d No", *voting.Yes, *voting.No)
				if voting.Abstain != nil {
					voteDetails += fmt.Sprintf(", %d Abstain", *voting.Abstain)
				}
				voteDetails += ")"
			}

			// Check if this might be related to legislation (basic heuristic)
			legislationHint := ""
			if voting.Topic != nil {
				topic := strings.ToLower(*voting.Topic)
				if strings.Contains(topic, "ustaw") || strings.Contains(topic, "kodeks") || strings.Contains(topic, "projekt") {
					legislationHint = "\n    💡 This vote may relate to legislation - search ELI database for related legal acts"
				}
			}

			searchSummary += fmt.Sprintf("- %s\n  %s - %s %s%s\n\n", title, date, result, voteDetails, legislationHint)
		}

		if len(allMatchingVotings) > 15 {
			searchSummary += fmt.Sprintf("... and %d more voting records.\n", len(allMatchingVotings)-15)
		}
	}

	return mcp.NewToolResultText(searchSummary), nil
}

func (s *SejmServer) handleGetTerms(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	endpoint := fmt.Sprintf("%s/sejm/term", sejmBaseURL)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve terms from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var terms []sejm.Term
	if err := json.Unmarshal(data, &terms); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse terms data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	summary := "Polish Parliament (Sejm) Terms:\n\n"
	for _, term := range terms {
		summary += fmt.Sprintf("Term %d:\n", term.Num)
		if term.From != nil {
			summary += fmt.Sprintf("  From: %s\n", term.From.Format("2006-01-02"))
		}
		if term.To != nil {
			summary += fmt.Sprintf("  To: %s\n", term.To.Format("2006-01-02"))
		}
		if term.Current != nil {
			summary += fmt.Sprintf("  Current: %t\n\n", *term.Current)
		} else {
			summary += "  Current: unknown\n\n"
		}
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetClubs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/clubs", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve clubs from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var clubs []sejm.Club
	if err := json.Unmarshal(data, &clubs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse clubs data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	summary := fmt.Sprintf("Parliamentary Clubs for Term %d:\n\n", term)
	for _, club := range clubs {
		if club.Name != nil {
			summary += fmt.Sprintf("• %s", *club.Name)
		}
		if club.Id != nil {
			summary += fmt.Sprintf(" (ID: %s)", *club.Id)
		}
		if club.MembersCount != nil {
			summary += fmt.Sprintf(" - %d members", *club.MembersCount)
		}
		summary += "\n"
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetVotingDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	sitting := request.GetString("sitting", "")
	votingNumber := request.GetString("voting_number", "")
	format := request.GetString("format", "json")

	if sitting == "" || votingNumber == "" {
		return mcp.NewToolResultError("Both 'sitting' and 'voting_number' parameters are required. Get these from sejm_search_votings results."), nil
	}

	// First get the detailed voting information (JSON)
	endpoint := fmt.Sprintf("%s/sejm/term%d/votings/%s/%s", sejmBaseURL, term, sitting, votingNumber)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve voting details: %v. Please verify sitting=%s and voting_number=%s exist.", err, sitting, votingNumber)), nil
	}

	var voting sejm.Voting
	if err := json.Unmarshal(data, &voting); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse voting data: %v.", err)), nil
	}

	if format == "json" {
		// Return structured JSON data
		result, _ := json.MarshalIndent(voting, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Detailed voting information for sitting %s, vote %s:\n\n%s", sitting, votingNumber, string(result))), nil
	}

	// For text/pdf formats, try to get the PDF version
	pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/votings/%s/%s/pdf", sejmBaseURL, term, sitting, votingNumber)

	if format == "pdf" {
		// Return PDF download info
		return mcp.NewToolResultText(fmt.Sprintf("PDF document available at: %s\n\nUse format='text' to get searchable text extracted from this PDF.", pdfEndpoint)), nil
	}

	if format == "text" {
		// Download PDF and convert to text
		pdfData, err := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for text conversion: %v. This voting may not have a PDF version available.", err)), nil
		}

		extractedText, err := s.extractTextFromPDF(pdfData)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to extract text from PDF: %v.", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Voting details for sitting %s, vote %s (converted from PDF):\n\n%s", sitting, votingNumber, extractedText)), nil
	}

	return mcp.NewToolResultError(fmt.Sprintf("Invalid format '%s'. Use 'json', 'text', or 'pdf'.", format)), nil
}

func (s *SejmServer) handleSearchVotingContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	sitting := request.GetString("sitting", "")
	votingNumber := request.GetString("voting_number", "")
	searchTerms := request.GetString("search_terms", "")
	contextChars := request.GetString("context_chars", "100")
	maxMatchesPerTerm := request.GetString("max_matches_per_term", "10")

	if sitting == "" || votingNumber == "" || searchTerms == "" {
		return mcp.NewToolResultError("Parameters 'sitting', 'voting_number', and 'search_terms' are all required."), nil
	}

	// Parse parameters similar to eli_search_act_content
	contextCharsInt := 100
	if contextChars != "" {
		if parsed, err := fmt.Sscanf(contextChars, "%d", &contextCharsInt); parsed == 1 && err == nil {
			if contextCharsInt > 500 {
				contextCharsInt = 500
			} else if contextCharsInt < 20 {
				contextCharsInt = 20
			}
		}
	}

	maxMatchesInt := 10
	if maxMatchesPerTerm != "" {
		if parsed, err := fmt.Sscanf(maxMatchesPerTerm, "%d", &maxMatchesInt); parsed == 1 && err == nil {
			if maxMatchesInt > 50 {
				maxMatchesInt = 50
			} else if maxMatchesInt < 1 {
				maxMatchesInt = 1
			}
		}
	}

	// Download the PDF
	pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/votings/%s/%s/pdf", sejmBaseURL, term, sitting, votingNumber)
	pdfData, err := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for search: %v. This voting may not have a PDF version available.", err)), nil
	}

	// Use the same search logic as ELI content search
	return s.searchPDFContent(ctx, pdfData, fmt.Sprintf("voting %s/%s", sitting, votingNumber), searchTerms, contextCharsInt, maxMatchesInt)
}

func (s *SejmServer) handleGetProceedings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	limit := request.GetString("limit", "20")
	params["limit"] = limit

	endpoint := fmt.Sprintf("%s/sejm/term%d/proceedings", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve proceedings from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var proceedings []sejm.Proceeding
	if err := json.Unmarshal(data, &proceedings); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse proceedings data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Sort proceedings by number in descending order to show most recent first
	for i := 0; i < len(proceedings)-1; i++ {
		for j := i + 1; j < len(proceedings); j++ {
			// Handle nil proceeding numbers safely
			numI := int32(0)
			if proceedings[i].Number != nil {
				numI = *proceedings[i].Number
			}
			numJ := int32(0)
			if proceedings[j].Number != nil {
				numJ = *proceedings[j].Number
			}
			// Sort in descending order (most recent first)
			if numI < numJ {
				proceedings[i], proceedings[j] = proceedings[j], proceedings[i]
			}
		}
	}

	summary := fmt.Sprintf("Parliamentary Proceedings for Term %d (most recent first):\n\n", term)
	for i, proceeding := range proceedings {
		if i >= 20 { // Limit displayed entries
			summary += fmt.Sprintf("... and %d more proceedings\n", len(proceedings)-i)
			break
		}

		if proceeding.Number != nil {
			summary += fmt.Sprintf("Proceeding %d:\n", *proceeding.Number)
		}
		if proceeding.Dates != nil && len(*proceeding.Dates) > 0 {
			summary += fmt.Sprintf("  Date: %s\n", (*proceeding.Dates)[0])
		}
		if proceeding.Title != nil {
			summary += fmt.Sprintf("  Title: %s\n", *proceeding.Title)
		}
		summary += "\n"
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetPrints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	limit := request.GetString("limit", "30")
	params["limit"] = limit

	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}
	if sortBy := request.GetString("sort_by", ""); sortBy != "" {
		params["sort_by"] = sortBy
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/prints", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve prints from Polish Parliament API: %v. Please try again.", err)), nil
	}

	var prints []sejm.Print
	if err := json.Unmarshal(data, &prints); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse prints data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	summary := fmt.Sprintf("Parliamentary Prints (Legislative Documents) for Term %d:\n\n", term)

	// Note: Print type doesn't have DocumentType field, so we'll just show the prints directly

	summary += "Recent Prints:\n"
	for i, printItem := range prints {
		if i >= 15 { // Limit displayed entries
			summary += fmt.Sprintf("... and %d more prints\n", len(prints)-i)
			break
		}

		if printItem.Number != nil {
			summary += fmt.Sprintf("Print %s:", *printItem.Number)
		}
		if printItem.Title != nil {
			summary += fmt.Sprintf(" %s", *printItem.Title)
		}
		summary += "\n"
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetTranscripts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	proceedingID := request.GetString("proceeding_id", "")
	date := request.GetString("date", "")
	format := request.GetString("format", "list")
	page := request.GetString("page", "1")
	pagesPerChunk := request.GetString("pages_per_chunk", "5")
	showPageInfo := request.GetString("show_page_info", "false")

	if proceedingID == "" || date == "" {
		return mcp.NewToolResultError("Both 'proceeding_id' and 'date' parameters are required. Get these from sejm_get_proceedings results."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/proceedings/%s/%s/transcripts", sejmBaseURL, term, proceedingID, date)

	if format == "pdf" {
		// Return PDF download info
		pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/proceedings/%s/%s/transcripts/pdf", sejmBaseURL, term, proceedingID, date)
		return mcp.NewToolResultText(fmt.Sprintf("PDF transcript available at: %s\n\nUse format='text' to get searchable text extracted from this PDF.", pdfEndpoint)), nil
	}

	if format == "text" {
		// Download PDF and convert to text with pagination
		pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/proceedings/%s/%s/transcripts/pdf", sejmBaseURL, term, proceedingID, date)
		pdfData, err := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for text conversion: %v. This proceeding may not have a PDF transcript available.", err)), nil
		}

		// Use pagination to manage large transcript responses
		return s.extractTextWithPagination(ctx, pdfData, "", "", fmt.Sprintf("proceeding-%s-%s", proceedingID, date), page, pagesPerChunk, showPageInfo)
	}

	// Default: return statement list
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve transcripts from Polish Parliament API: %v. Please verify proceeding_id=%s and date=%s exist.", err, proceedingID, date)), nil
	}

	var statements sejm.StatementList
	if err := json.Unmarshal(data, &statements); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse transcript data from API response: %v. The API may have returned unexpected data format.", err)), nil
	}

	// Create summary of statements
	summary := fmt.Sprintf("Parliamentary proceedings transcript for term %d, proceeding %s on %s:\n", term, proceedingID, date)
	summary += fmt.Sprintf("- Total statements: %d\n\n", len(*statements.Statements))

	summary += "Statement list (number, speaker, function, time):\n"
	for i, stmt := range *statements.Statements {
		if i >= 20 { // Limit displayed entries
			summary += fmt.Sprintf("... and %d more statements. Use sejm_get_statement with specific statement numbers for full content.\n", len(*statements.Statements)-i)
			break
		}

		speaker := "Unknown speaker"
		if stmt.Name != nil {
			speaker = *stmt.Name
		}

		function := ""
		if stmt.Function != nil {
			function = fmt.Sprintf(" (%s)", *stmt.Function)
		}

		time := ""
		if stmt.StartDateTime != nil {
			time = fmt.Sprintf(" - %s", stmt.StartDateTime.Format("15:04"))
		}

		summary += fmt.Sprintf("- Statement %d: %s%s%s\n", *stmt.Num, speaker, function, time)
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetStatement(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	proceedingID := request.GetString("proceeding_id", "")
	date := request.GetString("date", "")
	statementNum := request.GetString("statement_num", "")
	chunkSize := request.GetString("chunk_size", "5000")
	chunkNumber := request.GetString("chunk_number", "1")
	showChunkInfo := request.GetString("show_chunk_info", "false")

	if proceedingID == "" || date == "" || statementNum == "" {
		return mcp.NewToolResultError("Parameters 'proceeding_id', 'date', and 'statement_num' are all required. Get these from sejm_get_transcripts results."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/proceedings/%s/%s/transcripts/%s", sejmBaseURL, term, proceedingID, date, statementNum)
	data, err := s.makeTextRequest(ctx, endpoint, "html")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve statement from Polish Parliament API: %v. Please verify proceeding_id=%s, date=%s, and statement_num=%s exist.", err, proceedingID, date, statementNum)), nil
	}

	// Handle HTML chunking for large responses
	return s.chunkHTMLContent(string(data), fmt.Sprintf("Statement %s from proceeding %s on %s", statementNum, proceedingID, date), chunkSize, chunkNumber, showChunkInfo)
}

func (s *SejmServer) chunkHTMLContent(htmlContent, documentTitle, chunkSizeStr, chunkNumberStr, showChunkInfo string) (*mcp.CallToolResult, error) {
	// Parse parameters
	chunkSize := 5000
	if chunkSizeStr != "" {
		if parsed, err := fmt.Sscanf(chunkSizeStr, "%d", &chunkSize); parsed == 1 && err == nil {
			if chunkSize < 1000 {
				chunkSize = 1000
			} else if chunkSize > 10000 {
				chunkSize = 10000
			}
		}
	}

	chunkNumber := 1
	if chunkNumberStr != "" {
		if parsed, err := fmt.Sscanf(chunkNumberStr, "%d", &chunkNumber); parsed != 1 || err != nil || chunkNumber < 1 {
			chunkNumber = 1
		}
	}

	// Calculate total chunks
	totalChunks := (len(htmlContent) + chunkSize - 1) / chunkSize

	if showChunkInfo == "true" {
		chunkInfo := fmt.Sprintf("%s - Document Structure:\n", documentTitle)
		chunkInfo += fmt.Sprintf("- Total characters: %d\n", len(htmlContent))
		chunkInfo += fmt.Sprintf("- Chunk size: %d characters\n", chunkSize)
		chunkInfo += fmt.Sprintf("- Total chunks: %d\n\n", totalChunks)

		chunkInfo += "Navigation:\n"
		if chunkNumber > 1 {
			chunkInfo += fmt.Sprintf("- Previous chunk: chunk_number='%d'\n", chunkNumber-1)
		}
		if chunkNumber < totalChunks {
			chunkInfo += fmt.Sprintf("- Next chunk: chunk_number='%d'\n", chunkNumber+1)
		}
		chunkInfo += "- First chunk: chunk_number='1'\n"
		chunkInfo += fmt.Sprintf("- Last chunk: chunk_number='%d'\n", totalChunks)

		return mcp.NewToolResultText(chunkInfo), nil
	}

	// Validate chunk number
	if chunkNumber > totalChunks {
		return mcp.NewToolResultError(fmt.Sprintf("Chunk number %d exceeds total chunks %d. Use show_chunk_info='true' to see available chunks.", chunkNumber, totalChunks)), nil
	}

	// Extract the requested chunk
	startPos := (chunkNumber - 1) * chunkSize
	endPos := startPos + chunkSize
	if endPos > len(htmlContent) {
		endPos = len(htmlContent)
	}

	chunk := htmlContent[startPos:endPos]

	// Build response with navigation info
	response := fmt.Sprintf("%s (chunk %d/%d):\n\n", documentTitle, chunkNumber, totalChunks)
	response += chunk

	if totalChunks > 1 {
		response += fmt.Sprintf("\n\n--- Chunk %d of %d (characters %d-%d) ---\n", chunkNumber, totalChunks, startPos+1, endPos)
		if chunkNumber < totalChunks {
			response += fmt.Sprintf("Next chunk: chunk_number='%d'\n", chunkNumber+1)
		}
		if chunkNumber > 1 {
			response += fmt.Sprintf("Previous chunk: chunk_number='%d'\n", chunkNumber-1)
		}
		response += "Show structure: show_chunk_info='true'"
	}

	return mcp.NewToolResultText(response), nil
}

func (s *SejmServer) handleGetCommitteeSittingsByDate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	date := request.GetString("date", "")
	canceled := request.GetString("canceled", "false")

	if date == "" {
		return mcp.NewToolResultError("Date parameter is required in YYYY-MM-DD format (e.g., '2023-11-20')."), nil
	}

	params := make(map[string]string)
	if canceled == "true" {
		params["canceled"] = "true"
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/committees/sittings/%s", sejmBaseURL, term, date)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve committee sittings for date %s: %v. Please verify the date format is YYYY-MM-DD.", date, err)), nil
	}

	var sittings []sejm.CommitteeSitting
	if err := json.Unmarshal(data, &sittings); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse committee sittings data: %v.", err)), nil
	}

	summary := fmt.Sprintf("Committee meetings scheduled for %s (term %d):\n", date, term)
	summary += fmt.Sprintf("- Total meetings: %d\n\n", len(sittings))

	if len(sittings) == 0 {
		summary += "No committee meetings scheduled for this date.\n"
		return mcp.NewToolResultText(summary), nil
	}

	summary += "Meetings:\n"
	for i, sitting := range sittings {
		if i >= 15 { // Limit display
			summary += fmt.Sprintf("... and %d more meetings\n", len(sittings)-i)
			break
		}

		if sitting.Code != nil {
			summary += fmt.Sprintf("- Committee %s", *sitting.Code)
		}
		if sitting.Num != nil {
			summary += fmt.Sprintf(" (Meeting #%d)", *sitting.Num)
		}
		if sitting.StartDateTime != nil {
			summary += fmt.Sprintf(" - %s", sitting.StartDateTime.Format("15:04"))
		}
		if sitting.Room != nil {
			summary += fmt.Sprintf(" in %s", *sitting.Room)
		}
		summary += "\n"
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetCommitteeSittings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	committeeCode := request.GetString("committee_code", "")
	canceled := request.GetString("canceled", "false")

	if committeeCode == "" {
		return mcp.NewToolResultError("Committee code is required (e.g., 'ENM', 'ASW'). Get committee codes from sejm_get_committees."), nil
	}

	params := make(map[string]string)
	if canceled == "true" {
		params["canceled"] = "true"
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/committees/%s/sittings", sejmBaseURL, term, committeeCode)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve sittings for committee %s: %v. Please verify the committee code exists.", committeeCode, err)), nil
	}

	var sittings []sejm.CommitteeSitting
	if err := json.Unmarshal(data, &sittings); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse committee sittings data: %v.", err)), nil
	}

	summary := fmt.Sprintf("Committee %s meetings (term %d):\n", committeeCode, term)
	summary += fmt.Sprintf("- Total meetings: %d\n\n", len(sittings))

	if len(sittings) == 0 {
		summary += "No meetings found for this committee.\n"
		return mcp.NewToolResultText(summary), nil
	}

	summary += "Recent meetings:\n"
	for i, sitting := range sittings {
		if i >= 20 { // Limit display
			summary += fmt.Sprintf("... and %d more meetings\n", len(sittings)-i)
			break
		}

		if sitting.Num != nil {
			summary += fmt.Sprintf("- Meeting #%d", *sitting.Num)
		}
		if sitting.Date != nil {
			summary += fmt.Sprintf(" on %s", sitting.Date.Format("2006-01-02"))
		}
		if sitting.StartDateTime != nil && sitting.EndDateTime != nil {
			summary += fmt.Sprintf(" (%s-%s)",
				sitting.StartDateTime.Format("15:04"),
				sitting.EndDateTime.Format("15:04"))
		}
		summary += "\n"
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetCommitteeSittingDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	committeeCode := request.GetString("committee_code", "")
	sittingNumber := request.GetString("sitting_number", "")

	if committeeCode == "" || sittingNumber == "" {
		return mcp.NewToolResultError("Both committee_code and sitting_number are required. Get these from committee sitting lists."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/committees/%s/sittings/%s", sejmBaseURL, term, committeeCode, sittingNumber)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve committee sitting details: %v. Please verify committee_code=%s and sitting_number=%s exist.", err, committeeCode, sittingNumber)), nil
	}

	var sitting sejm.CommitteeSitting
	if err := json.Unmarshal(data, &sitting); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse committee sitting data: %v.", err)), nil
	}

	summary := fmt.Sprintf("Committee %s Meeting #%s Details:\n\n", committeeCode, sittingNumber)

	if sitting.Date != nil {
		summary += fmt.Sprintf("Date: %s\n", sitting.Date.Format("2006-01-02"))
	}
	if sitting.StartDateTime != nil {
		summary += fmt.Sprintf("Start Time: %s\n", sitting.StartDateTime.Format("15:04"))
	}
	if sitting.EndDateTime != nil {
		summary += fmt.Sprintf("End Time: %s\n", sitting.EndDateTime.Format("15:04"))
	}
	if sitting.Room != nil {
		summary += fmt.Sprintf("Room: %s\n", *sitting.Room)
	}
	if sitting.Closed != nil {
		status := "Open"
		if *sitting.Closed {
			status = "Closed"
		}
		summary += fmt.Sprintf("Status: %s\n", status)
	}

	if sitting.Agenda != nil && *sitting.Agenda != "" {
		summary += fmt.Sprintf("\nAgenda:\n%s\n", *sitting.Agenda)
	}

	if sitting.Notes != nil && *sitting.Notes != "" {
		summary += fmt.Sprintf("\nNotes: %s\n", *sitting.Notes)
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetCommitteeTranscript(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	committeeCode := request.GetString("committee_code", "")
	sittingNumber := request.GetString("sitting_number", "")
	format := request.GetString("format", "html")

	// PDF pagination parameters
	page := request.GetString("page", "1")
	pagesPerChunk := request.GetString("pages_per_chunk", "5")
	showPageInfo := request.GetString("show_page_info", "false")

	// HTML chunking parameters
	chunkSize := request.GetString("chunk_size", "5000")
	chunkNumber := request.GetString("chunk_number", "1")
	showChunkInfo := request.GetString("show_chunk_info", "false")

	if committeeCode == "" || sittingNumber == "" {
		return mcp.NewToolResultError("Both committee_code and sitting_number are required. Get these from committee sitting lists."), nil
	}

	if format == "pdf" {
		// Return PDF download info
		pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/committees/%s/sittings/%s/pdf", sejmBaseURL, term, committeeCode, sittingNumber)
		return mcp.NewToolResultText(fmt.Sprintf("Committee transcript PDF available at: %s\n\nUse format='text' to get searchable text extracted from this PDF with pagination support.", pdfEndpoint)), nil
	}

	if format == "text" {
		// Download PDF and convert to text with pagination
		pdfEndpoint := fmt.Sprintf("%s/sejm/term%d/committees/%s/sittings/%s/pdf", sejmBaseURL, term, committeeCode, sittingNumber)
		pdfData, err := s.makeTextRequest(ctx, pdfEndpoint, "pdf")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve PDF for text conversion: %v. This committee meeting may not have a PDF transcript available.", err)), nil
		}

		// Use pagination to manage large transcript responses
		return s.extractTextWithPagination(ctx, pdfData, "", "", fmt.Sprintf("committee-%s-sitting-%s", committeeCode, sittingNumber), page, pagesPerChunk, showPageInfo)
	}

	// Default: HTML format with chunking
	htmlEndpoint := fmt.Sprintf("%s/sejm/term%d/committees/%s/sittings/%s/html", sejmBaseURL, term, committeeCode, sittingNumber)
	htmlData, err := s.makeTextRequest(ctx, htmlEndpoint, "html")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve HTML transcript: %v. This committee meeting may not have an HTML transcript available.", err)), nil
	}

	// Handle HTML chunking for large responses
	documentTitle := fmt.Sprintf("Committee %s Meeting #%s Transcript", committeeCode, sittingNumber)
	return s.chunkHTMLContent(string(htmlData), documentTitle, chunkSize, chunkNumber, showChunkInfo)
}

func (s *SejmServer) handleGetMPPhoto(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	mpID := request.GetString("mp_id", "")
	if mpID == "" {
		return mcp.NewToolResultError("MP ID is required. Please provide the mp_id parameter with a valid MP identification number. You can get MP IDs from the sejm_get_mps tool."), nil
	}

	size := request.GetString("size", "full")

	var endpoint string
	if size == "mini" {
		endpoint = fmt.Sprintf("%s/sejm/term%d/MP/%s/photo-mini", sejmBaseURL, term, mpID)
	} else {
		endpoint = fmt.Sprintf("%s/sejm/term%d/MP/%s/photo", sejmBaseURL, term, mpID)
	}

	// Make request for image data
	imageData, err := s.makeTextRequest(ctx, endpoint, "image")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve MP photo: %v. Please verify the MP ID (%s) exists in term %d and has a photo available.", err, mpID, term)), nil
	}

	photoSize := "full size"
	if size == "mini" {
		photoSize = "mini (thumbnail)"
	}

	return mcp.NewToolResultText(fmt.Sprintf("MP photo for ID %s (term %d) retrieved successfully in %s format.\n\nPhoto data: %d bytes\nEndpoint: %s\n\nNote: This is binary image data (JPEG format). The photo shows the official parliamentary portrait of the MP used in parliamentary documentation and public materials.", mpID, term, photoSize, len(imageData), endpoint)), nil
}

func (s *SejmServer) handleGetMPVotingStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	mpID := request.GetString("mp_id", "")
	if mpID == "" {
		return mcp.NewToolResultError("MP ID is required. Please provide the mp_id parameter with a valid MP identification number. You can get MP IDs from the sejm_get_mps tool."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/MP/%s/votings/stats", sejmBaseURL, term, mpID)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve MP voting statistics: %v. Please verify the MP ID (%s) exists in term %d.", err, mpID, term)), nil
	}

	var stats []sejm.VotingStat
	if err := json.Unmarshal(data, &stats); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse voting statistics data: %v.", err)), nil
	}

	// Analyze voting statistics
	totalSittings := len(stats)
	totalVotings := 0
	totalVoted := 0
	totalMissed := 0
	sittingsWithExcuse := 0

	for _, stat := range stats {
		if stat.NumVotings != nil {
			totalVotings += int(*stat.NumVotings)
		}
		if stat.NumVoted != nil {
			totalVoted += int(*stat.NumVoted)
		}
		if stat.NumMissed != nil {
			totalMissed += int(*stat.NumMissed)
		}
		if stat.AbsenceExcuse != nil && *stat.AbsenceExcuse {
			sittingsWithExcuse++
		}
	}

	// Calculate participation rates
	participationRate := 0.0
	if totalVotings > 0 {
		participationRate = float64(totalVoted) / float64(totalVotings) * 100
	}

	attendanceRate := 0.0
	if totalSittings > 0 {
		sittingsAttended := totalSittings - sittingsWithExcuse
		attendanceRate = float64(sittingsAttended) / float64(totalSittings) * 100
	}

	summary := fmt.Sprintf("Voting statistics for MP %s (term %d):\n\n", mpID, term)
	summary += "Overall Performance:\n"
	summary += fmt.Sprintf("- Parliamentary sittings tracked: %d\n", totalSittings)
	summary += fmt.Sprintf("- Total voting opportunities: %d\n", totalVotings)
	summary += fmt.Sprintf("- Votes cast: %d\n", totalVoted)
	summary += fmt.Sprintf("- Votes missed: %d\n", totalMissed)
	summary += fmt.Sprintf("- Voting participation rate: %.1f%%\n", participationRate)
	summary += fmt.Sprintf("- Sitting attendance rate: %.1f%%\n", attendanceRate)
	summary += fmt.Sprintf("- Sittings with absence excuse: %d\n\n", sittingsWithExcuse)

	summary += "Recent sitting breakdown (last 10 sittings):\n"
	recentCount := 10
	if len(stats) < recentCount {
		recentCount = len(stats)
	}

	for i := len(stats) - recentCount; i < len(stats); i++ {
		stat := stats[i]
		date := "Unknown date"
		if stat.Date != nil {
			date = stat.Date.Format("2006-01-02")
		}

		sitting := "Unknown"
		if stat.Sitting != nil {
			sitting = fmt.Sprintf("%d", *stat.Sitting)
		}

		voted := 0
		if stat.NumVoted != nil {
			voted = int(*stat.NumVoted)
		}

		votings := 0
		if stat.NumVotings != nil {
			votings = int(*stat.NumVotings)
		}

		missed := 0
		if stat.NumMissed != nil {
			missed = int(*stat.NumMissed)
		}

		excuse := ""
		if stat.AbsenceExcuse != nil && *stat.AbsenceExcuse {
			excuse = " (excused absence)"
		}

		summary += fmt.Sprintf("- Sitting %s (%s): %d/%d votes cast, %d missed%s\n", sitting, date, voted, votings, missed, excuse)
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetMPVotingDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	mpID := request.GetString("mp_id", "")
	sitting := request.GetString("sitting", "")
	date := request.GetString("date", "")

	if mpID == "" || sitting == "" || date == "" {
		return mcp.NewToolResultError("All parameters are required: mp_id, sitting, and date. Get sitting numbers from sejm_search_votings or sejm_get_proceedings results."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/MP/%s/votings/%s/%s", sejmBaseURL, term, mpID, sitting, date)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve MP voting details: %v. Please verify MP ID (%s), sitting (%s), and date (%s) are correct.", err, mpID, sitting, date)), nil
	}

	var votes []sejm.VoteMP
	if err := json.Unmarshal(data, &votes); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse voting details data: %v.", err)), nil
	}

	if len(votes) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No voting records found for MP %s during sitting %s on %s (term %d). This could mean:\n- The MP was not present during this sitting\n- No votes were cast during this sitting\n- The sitting/date combination is invalid", mpID, sitting, date, term)), nil
	}

	// Analyze voting patterns
	yesVotes := 0
	noVotes := 0
	abstainVotes := 0
	absentVotes := 0
	otherVotes := 0

	for _, vote := range votes {
		if vote.Vote != nil {
			switch *vote.Vote {
			case "YES":
				yesVotes++
			case "NO":
				noVotes++
			case "ABSTAIN":
				abstainVotes++
			case "ABSENT":
				absentVotes++
			default:
				otherVotes++
			}
		}
	}

	summary := fmt.Sprintf("Detailed voting record for MP %s during sitting %s on %s (term %d):\n\n", mpID, sitting, date, term)
	summary += "Voting Summary:\n"
	summary += fmt.Sprintf("- Total votes: %d\n", len(votes))
	summary += fmt.Sprintf("- Yes votes: %d\n", yesVotes)
	summary += fmt.Sprintf("- No votes: %d\n", noVotes)
	summary += fmt.Sprintf("- Abstain votes: %d\n", abstainVotes)
	summary += fmt.Sprintf("- Absent/No vote: %d\n", absentVotes)
	if otherVotes > 0 {
		summary += fmt.Sprintf("- Other: %d\n", otherVotes)
	}
	summary += "\nDetailed vote-by-vote record:\n"

	// Show first 15 votes to avoid overwhelming output
	displayCount := 15
	if len(votes) < displayCount {
		displayCount = len(votes)
	}

	for i := 0; i < displayCount; i++ {
		vote := votes[i]

		title := "No title"
		if vote.Title != nil {
			title = *vote.Title
		}

		topic := ""
		if vote.Topic != nil && *vote.Topic != "" {
			topic = fmt.Sprintf(" (Topic: %s)", *vote.Topic)
		}

		voteValue := "Unknown"
		if vote.Vote != nil {
			voteValue = string(*vote.Vote)
		}

		time := ""
		if vote.Date != nil {
			time = fmt.Sprintf(" - %s", vote.Date.Format("15:04"))
		}

		votingNum := ""
		if vote.VotingNumber != nil {
			votingNum = fmt.Sprintf("Vote #%d: ", *vote.VotingNumber)
		}

		summary += fmt.Sprintf("- %s%s → %s%s%s\n", votingNum, title, voteValue, topic, time)
	}

	if len(votes) > displayCount {
		summary += fmt.Sprintf("\n... and %d more votes. Total voting record shows %d legislative decisions.", len(votes)-displayCount, len(votes))
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetVideos(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)

	if committee := request.GetString("committee", ""); committee != "" {
		params["comm"] = committee
	}
	if since := request.GetString("since", ""); since != "" {
		params["since"] = since
	}
	if till := request.GetString("till", ""); till != "" {
		params["till"] = till
	}
	if title := request.GetString("title", ""); title != "" {
		params["title"] = title
	}
	if videoType := request.GetString("type", ""); videoType != "" {
		params["type"] = videoType
	}
	if limit := request.GetString("limit", ""); limit != "" {
		params["limit"] = limit
	} else {
		params["limit"] = "20" // Default limit
	}
	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/videos", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve videos: %v. Please try again.", err)), nil
	}

	var videos []sejm.Video
	if err := json.Unmarshal(data, &videos); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse videos data: %v.", err)), nil
	}

	if len(videos) == 0 {
		filtersUsed := []string{}
		if params["comm"] != "" {
			filtersUsed = append(filtersUsed, fmt.Sprintf("committee: %s", params["comm"]))
		}
		if params["since"] != "" {
			filtersUsed = append(filtersUsed, fmt.Sprintf("from: %s", params["since"]))
		}
		if params["till"] != "" {
			filtersUsed = append(filtersUsed, fmt.Sprintf("until: %s", params["till"]))
		}
		if params["title"] != "" {
			filtersUsed = append(filtersUsed, fmt.Sprintf("title: %s", params["title"]))
		}
		if params["type"] != "" {
			filtersUsed = append(filtersUsed, fmt.Sprintf("type: %s", params["type"]))
		}

		filterText := ""
		if len(filtersUsed) > 0 {
			filterText = fmt.Sprintf(" with filters: %s", strings.Join(filtersUsed, ", "))
		}

		return mcp.NewToolResultText(fmt.Sprintf("No video transmissions found for term %d%s.\n\nTry:\n- Removing or adjusting date filters\n- Using different committee codes\n- Searching with broader title terms\n- Checking if videos exist for this term period", term, filterText)), nil
	}

	// Analyze video types and content
	liveCount := 0
	committeeCount := 0
	plenaryCount := 0

	for _, video := range videos {
		if video.Type != nil {
			switch strings.ToLower(*video.Type) {
			case "komisja", "committee":
				committeeCount++
			case "posiedzenie", "plenary":
				plenaryCount++
			}
		}
		// Check if it's currently live (simplified check)
		if video.StartDateTime != nil && video.EndDateTime == nil {
			liveCount++
		}
	}

	summary := fmt.Sprintf("Video transmissions for parliamentary term %d:\n", term)
	summary += fmt.Sprintf("- Total videos found: %d\n", len(videos))
	summary += fmt.Sprintf("- Committee meetings: %d\n", committeeCount)
	summary += fmt.Sprintf("- Plenary sessions: %d\n", plenaryCount)
	summary += fmt.Sprintf("- Currently live: %d\n\n", liveCount)

	// Show recent videos (limit to 10 for readability)
	displayCount := 10
	if len(videos) < displayCount {
		displayCount = len(videos)
	}

	summary += "Recent video transmissions:\n"
	for i := 0; i < displayCount; i++ {
		video := videos[i]

		title := "No title"
		if video.Title != nil {
			title = *video.Title
		}

		videoType := "Unknown type"
		if video.Type != nil {
			videoType = *video.Type
		}

		room := ""
		if video.Room != nil {
			room = fmt.Sprintf(" in %s", *video.Room)
		}

		committee := ""
		if video.Committee != nil {
			committee = fmt.Sprintf(" (Committee: %s)", *video.Committee)
		}

		startTime := ""
		if video.StartDateTime != nil {
			startTime = fmt.Sprintf(" - %s", video.StartDateTime.Format("2006-01-02 15:04"))
		}

		streamingInfo := ""
		if video.VideoLink != nil {
			streamingInfo = " [Streaming available]"
		}

		unid := ""
		if video.Unid != nil {
			unid = fmt.Sprintf(" (ID: %s)", *video.Unid)
		}

		summary += fmt.Sprintf("- %s (%s)%s%s%s%s%s\n", title, videoType, room, committee, startTime, streamingInfo, unid)
	}

	if len(videos) > displayCount {
		summary += fmt.Sprintf("\n... and %d more videos. Use offset parameter for pagination or add filters to narrow results.", len(videos)-displayCount)
	}

	summary += "\n\nUse sejm_get_video_details with a specific video ID (unid) to get streaming URLs and detailed metadata."

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetVideosToday(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/videos/today", sejmBaseURL, term)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve today's videos: %v. Please try again.", err)), nil
	}

	var videos []sejm.Video
	if err := json.Unmarshal(data, &videos); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse today's videos data: %v.", err)), nil
	}

	today := time.Now().Format("2006-01-02")

	if len(videos) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No video transmissions scheduled for today (%s) in term %d.\n\nThis could mean:\n- No parliamentary sessions today\n- No committee meetings scheduled\n- Parliament is in recess\n- Technical issues with the streaming system", today, term)), nil
	}

	// Categorize today's videos
	liveNow := []sejm.Video{}
	upcoming := []sejm.Video{}
	completed := []sejm.Video{}
	currentTime := time.Now()

	for _, video := range videos {
		if video.StartDateTime != nil && video.EndDateTime != nil {
			if video.EndDateTime.Before(currentTime) {
				completed = append(completed, video)
			} else if video.StartDateTime.After(currentTime) {
				upcoming = append(upcoming, video)
			} else {
				liveNow = append(liveNow, video)
			}
		} else if video.StartDateTime != nil {
			if video.StartDateTime.Before(currentTime) {
				liveNow = append(liveNow, video) // Might still be live
			} else {
				upcoming = append(upcoming, video)
			}
		}
	}

	summary := fmt.Sprintf("Today's parliamentary video transmissions (%s, term %d):\n\n", today, term)
	summary += "📊 Overview:\n"
	summary += fmt.Sprintf("- Total transmissions: %d\n", len(videos))
	summary += fmt.Sprintf("- Currently live: %d\n", len(liveNow))
	summary += fmt.Sprintf("- Upcoming: %d\n", len(upcoming))
	summary += fmt.Sprintf("- Completed: %d\n\n", len(completed))

	if len(liveNow) > 0 {
		summary += "🔴 LIVE NOW:\n"
		for _, video := range liveNow {
			title := "No title"
			if video.Title != nil {
				title = *video.Title
			}

			room := ""
			if video.Room != nil {
				room = fmt.Sprintf(" in %s", *video.Room)
			}

			streamingInfo := ""
			if video.PlayerLink != nil {
				streamingInfo = " [Watch live]"
			}

			summary += fmt.Sprintf("- %s%s%s\n", title, room, streamingInfo)
		}
		summary += "\n"
	}

	if len(upcoming) > 0 {
		summary += "⏰ UPCOMING:\n"
		for _, video := range upcoming {
			title := "No title"
			if video.Title != nil {
				title = *video.Title
			}

			startTime := ""
			if video.StartDateTime != nil {
				startTime = fmt.Sprintf(" at %s", video.StartDateTime.Format("15:04"))
			}

			room := ""
			if video.Room != nil {
				room = fmt.Sprintf(" in %s", *video.Room)
			}

			summary += fmt.Sprintf("- %s%s%s\n", title, startTime, room)
		}
		summary += "\n"
	}

	if len(completed) > 0 {
		summary += "✅ COMPLETED TODAY:\n"
		for i, video := range completed {
			if i >= 5 { // Limit completed list
				summary += fmt.Sprintf("... and %d more completed transmissions\n", len(completed)-i)
				break
			}

			title := "No title"
			if video.Title != nil {
				title = *video.Title
			}

			timeRange := ""
			if video.StartDateTime != nil && video.EndDateTime != nil {
				timeRange = fmt.Sprintf(" (%s-%s)",
					video.StartDateTime.Format("15:04"),
					video.EndDateTime.Format("15:04"))
			}

			summary += fmt.Sprintf("- %s%s\n", title, timeRange)
		}
	}

	summary += "\nUse sejm_get_video_details with specific video IDs to access streaming URLs and detailed information."

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetVideosByDate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	date := request.GetString("date", "")
	if date == "" {
		return mcp.NewToolResultError("Date parameter is required in YYYY-MM-DD format (e.g., '2023-12-13')."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/videos/%s", sejmBaseURL, term, date)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve videos for date %s: %v. Please verify the date format is YYYY-MM-DD.", date, err)), nil
	}

	var videos []sejm.Video
	if err := json.Unmarshal(data, &videos); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse videos data for date %s: %v.", date, err)), nil
	}

	if len(videos) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No video transmissions found for %s (term %d).\n\nThis could mean:\n- No parliamentary sessions on this date\n- No committee meetings scheduled\n- Parliament was in recess\n- Date is outside the term period", date, term)), nil
	}

	// Analyze video types and timing
	committeeCount := 0
	plenaryCount := 0
	totalDuration := 0.0

	for _, video := range videos {
		if video.Type != nil {
			switch strings.ToLower(*video.Type) {
			case "komisja", "committee":
				committeeCount++
			case "posiedzenie", "plenary":
				plenaryCount++
			}
		}

		// Calculate duration if available
		if video.StartDateTime != nil && video.EndDateTime != nil {
			duration := video.EndDateTime.Sub(video.StartDateTime.Time)
			totalDuration += duration.Hours()
		}
	}

	summary := fmt.Sprintf("Parliamentary video transmissions for %s (term %d):\n\n", date, term)
	summary += "📊 Summary:\n"
	summary += fmt.Sprintf("- Total transmissions: %d\n", len(videos))
	summary += fmt.Sprintf("- Committee meetings: %d\n", committeeCount)
	summary += fmt.Sprintf("- Plenary sessions: %d\n", plenaryCount)
	if totalDuration > 0 {
		summary += fmt.Sprintf("- Total broadcast time: %.1f hours\n", totalDuration)
	}
	summary += "\n"

	summary += "📺 Transmissions:\n"
	for i, video := range videos {
		if i >= 15 { // Limit display
			summary += fmt.Sprintf("... and %d more transmissions\n", len(videos)-i)
			break
		}

		title := "No title"
		if video.Title != nil {
			title = *video.Title
		}

		videoType := ""
		if video.Type != nil {
			videoType = fmt.Sprintf(" (%s)", *video.Type)
		}

		timeInfo := ""
		if video.StartDateTime != nil {
			startTime := video.StartDateTime.Format("15:04")
			if video.EndDateTime != nil {
				endTime := video.EndDateTime.Format("15:04")
				duration := video.EndDateTime.Sub(video.StartDateTime.Time)
				timeInfo = fmt.Sprintf(" %s-%s (%.0f min)", startTime, endTime, duration.Minutes())
			} else {
				timeInfo = fmt.Sprintf(" from %s", startTime)
			}
		}

		room := ""
		if video.Room != nil {
			room = fmt.Sprintf(" in %s", *video.Room)
		}

		committee := ""
		if video.Committee != nil {
			committee = fmt.Sprintf(" [%s]", *video.Committee)
		}

		availableContent := ""
		if video.VideoLink != nil {
			availableContent = " 📹"
		}
		if video.Audio != nil {
			availableContent += " 🎵"
		}
		if video.SignLangLink != nil {
			availableContent += " 🤟"
		}

		unid := ""
		if video.Unid != nil {
			unid = fmt.Sprintf(" (ID: %s)", (*video.Unid)[:8]+"...") // Shortened ID
		}

		summary += fmt.Sprintf("- %s%s%s%s%s%s%s\n", title, videoType, timeInfo, room, committee, availableContent, unid)
	}

	summary += "\n📌 Use sejm_get_video_details with specific video IDs for streaming URLs and complete metadata."

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetVideoDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	unid := request.GetString("unid", "")
	if unid == "" {
		return mcp.NewToolResultError("Video ID (unid) is required. Get this from video listing results (32-character alphanumeric identifier)."), nil
	}

	endpoint := fmt.Sprintf("%s/sejm/term%d/videos/%s", sejmBaseURL, term, unid)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve video details for ID %s: %v. Please verify the video ID exists in term %d.", unid, err, term)), nil
	}

	var video sejm.Video
	if err := json.Unmarshal(data, &video); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse video details: %v.", err)), nil
	}

	summary := fmt.Sprintf("📺 Video Transmission Details (ID: %s, term %d):\n\n", unid, term)

	// Basic information
	if video.Title != nil {
		summary += fmt.Sprintf("🎬 Title: %s\n", *video.Title)
	}
	if video.Type != nil {
		summary += fmt.Sprintf("📋 Type: %s\n", *video.Type)
	}
	if video.Description != nil && *video.Description != "" {
		summary += fmt.Sprintf("📝 Description: %s\n", *video.Description)
	}

	// Location and committee info
	if video.Room != nil {
		summary += fmt.Sprintf("🏛️ Room: %s\n", *video.Room)
	}
	if video.Committee != nil {
		summary += fmt.Sprintf("👥 Committee: %s\n", *video.Committee)
	}
	if video.Subcommittee != nil {
		summary += fmt.Sprintf("👤 Subcommittee: %s\n", *video.Subcommittee)
	}

	// Timing information
	if video.StartDateTime != nil {
		summary += fmt.Sprintf("⏰ Start: %s\n", video.StartDateTime.Format("2006-01-02 15:04:05"))
	}
	if video.EndDateTime != nil {
		summary += fmt.Sprintf("⏰ End: %s\n", video.EndDateTime.Format("2006-01-02 15:04:05"))
		if video.StartDateTime != nil {
			duration := video.EndDateTime.Sub(video.StartDateTime.Time)
			summary += fmt.Sprintf("⏱️ Duration: %s\n", duration.String())
		}
	} else if video.StartDateTime != nil {
		summary += "🔴 Status: May still be live or ongoing\n"
	}

	summary += "\n🎥 Media Content:\n"

	// Video streaming links
	if video.VideoLink != nil {
		summary += fmt.Sprintf("📺 Main Video Stream: %s\n", *video.VideoLink)
	}

	// Multiple camera angles
	if video.OtherVideoLinks != nil && len(*video.OtherVideoLinks) > 0 {
		summary += "📹 Additional Camera Angles:\n"
		for i, link := range *video.OtherVideoLinks {
			summary += fmt.Sprintf("  Camera %d: %s\n", i+1, link)
		}
	}

	// Audio stream
	if video.Audio != nil {
		summary += fmt.Sprintf("🎵 Audio Stream: %s\n", *video.Audio)
	}

	// Sign language stream
	if video.SignLangLink != nil {
		summary += fmt.Sprintf("🤟 Sign Language Stream: %s\n", *video.SignLangLink)
	}

	// Player links
	summary += "\n🖥️ Web Players:\n"
	if video.PlayerLink != nil {
		summary += fmt.Sprintf("🔗 Sejm Player: %s\n", *video.PlayerLink)
	}
	if video.PlayerLinkIFrame != nil {
		summary += fmt.Sprintf("📺 Embeddable Player: %s\n", *video.PlayerLinkIFrame)
	}

	// Messages and interaction
	if video.VideoMessagesLink != nil {
		summary += fmt.Sprintf("💬 Messages/Chat: %s\n", *video.VideoMessagesLink)
	}

	// Transcription availability
	if video.Transcribe != nil {
		transcriptStatus := "No"
		if *video.Transcribe {
			transcriptStatus = "Yes"
		}
		summary += fmt.Sprintf("📄 Transcription Available: %s\n", transcriptStatus)
	}

	summary += "\n📋 Technical Details:\n"
	summary += fmt.Sprintf("🆔 Unique ID: %s\n", unid)
	if video.Type != nil {
		summary += fmt.Sprintf("🏷️ Transmission Type: %s\n", *video.Type)
	}

	// Summary of available media formats
	mediaFormats := []string{}
	if video.VideoLink != nil {
		mediaFormats = append(mediaFormats, "HLS Video")
	}
	if video.Audio != nil {
		mediaFormats = append(mediaFormats, "Audio")
	}
	if video.SignLangLink != nil {
		mediaFormats = append(mediaFormats, "Sign Language")
	}
	if video.OtherVideoLinks != nil && len(*video.OtherVideoLinks) > 0 {
		mediaFormats = append(mediaFormats, fmt.Sprintf("%d Camera Angles", len(*video.OtherVideoLinks)))
	}

	if len(mediaFormats) > 0 {
		summary += fmt.Sprintf("📊 Available Formats: %s\n", strings.Join(mediaFormats, ", "))
	}

	return mcp.NewToolResultText(summary), nil
}

func (s *SejmServer) handleGetWrittenQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	termStr := request.GetString("term", "10")
	term, err := s.validateTerm(termStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid term: %v", err)), nil
	}

	// Build API parameters
	params := make(map[string]string)

	if limit := request.GetString("limit", ""); limit != "" {
		params["limit"] = limit
	}
	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}
	if sortBy := request.GetString("sort_by", ""); sortBy != "" {
		params["sort_by"] = sortBy
	}
	if from := request.GetString("from", ""); from != "" {
		params["from"] = from
	}
	if to := request.GetString("to", ""); to != "" {
		params["to"] = to
	}
	if title := request.GetString("title", ""); title != "" {
		params["title"] = title
	}
	if since := request.GetString("since", ""); since != "" {
		params["since"] = since
	}
	if till := request.GetString("till", ""); till != "" {
		params["till"] = till
	}
	if delayed := request.GetString("delayed", ""); delayed != "" {
		params["delayed"] = delayed
	}

	s.logger.Info("sejm_get_written_questions called",
		slog.String("term", termStr),
		slog.Any("params", params))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/writtenQuestions", term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch written questions: %v", err)), nil
	}

	var questions []sejm.WrittenQuestion
	if err := json.Unmarshal(data, &questions); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse written questions: %v", err)), nil
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Term: %d", term))
	summary = append(summary, fmt.Sprintf("Found %d written questions", len(questions)))

	// Add filter info
	if from := request.GetString("from", ""); from != "" {
		summary = append(summary, fmt.Sprintf("From MP ID: %s", from))
	}
	if to := request.GetString("to", ""); to != "" {
		summary = append(summary, fmt.Sprintf("To: %s", to))
	}
	if title := request.GetString("title", ""); title != "" {
		summary = append(summary, fmt.Sprintf("Title filter: '%s'", title))
	}
	if delayed := request.GetString("delayed", ""); delayed == "true" {
		summary = append(summary, "Showing only delayed answers")
	}

	var results []string
	if len(questions) == 0 {
		results = append(results, "No written questions found matching the criteria.")
		results = append(results, "")
		results = append(results, "Try adjusting your search parameters:")
		results = append(results, "• Remove filters to see all questions")
		results = append(results, "• Check different time periods with 'since' and 'till'")
		results = append(results, "• Try different MP IDs with 'from' parameter")
	} else {
		// Show first 15 questions
		displayCount := 15
		if len(questions) < displayCount {
			displayCount = len(questions)
		}

		results = append(results, fmt.Sprintf("Showing first %d written questions:", displayCount))
		results = append(results, "")

		for i := 0; i < displayCount; i++ {
			q := questions[i]

			num := "Unknown"
			if q.Num != nil {
				num = fmt.Sprintf("%d", *q.Num)
			}

			title := "No title"
			if q.Title != nil {
				title = *q.Title
			}

			// Truncate long titles
			if len(title) > 80 {
				title = title[:77] + "..."
			}

			date := "Unknown date"
			if q.ReceiptDate != nil {
				date = q.ReceiptDate.String()
			}

			// Show sender info
			fromInfo := ""
			if q.From != nil && len(*q.From) > 0 {
				if len(*q.From) == 1 {
					fromInfo = fmt.Sprintf(" (from MP %s)", (*q.From)[0])
				} else {
					fromInfo = fmt.Sprintf(" (from %d MPs)", len(*q.From))
				}
			}

			// Show recipient info
			toInfo := ""
			if q.To != nil && len(*q.To) > 0 {
				if len(*q.To) == 1 {
					toInfo = fmt.Sprintf(" → %s", (*q.To)[0])
				} else {
					toInfo = fmt.Sprintf(" → %d recipients", len(*q.To))
				}
			}

			// Show delay info if available
			delayInfo := ""
			if q.AnswerDelayedDays != nil && *q.AnswerDelayedDays > 0 {
				delayInfo = fmt.Sprintf(" [DELAYED %d days]", *q.AnswerDelayedDays)
			}

			results = append(results, fmt.Sprintf("%s. %s%s%s%s", num, title, fromInfo, toInfo, delayInfo))
			results = append(results, fmt.Sprintf("   📅 %s", date))

			// Show replies count
			if q.Replies != nil {
				replyCount := len(*q.Replies)
				if replyCount > 0 {
					results = append(results, fmt.Sprintf("   💬 %d replies received", replyCount))
				} else {
					results = append(results, "   ⏳ No replies yet")
				}
			}

			results = append(results, "")
		}

		if len(questions) > displayCount {
			results = append(results, fmt.Sprintf("... and %d more questions available", len(questions)-displayCount))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "Filter by MP: use 'from' parameter with MP ID from sejm_get_mps")
	nextActions = append(nextActions, "Filter by ministry: use 'to' parameter with ministry name")
	nextActions = append(nextActions, "Find delayed answers: use delayed='true'")
	nextActions = append(nextActions, "Search by topic: use 'title' parameter with keywords")

	// Add pagination hints if we have results
	if len(questions) > 0 {
		if offset := request.GetString("offset", ""); offset == "" {
			nextActions = append(nextActions, "Next page: add offset='50' for more results")
		}
		if sortBy := request.GetString("sort_by", ""); sortBy == "" {
			nextActions = append(nextActions, "Sort by date: add sort_by='-receiptDate' for newest first")
		}
	}

	response := StandardResponse{
		Operation:   "Parliamentary Written Questions",
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Written questions (zapytania) are formal inquiries requiring government response within statutory timeframes. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) registerProcessesTools() {
	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_processes",
		Description: "Retrieve parliamentary legislative processes for a specific term. Returns comprehensive information about legislative procedures including bills, resolutions, and other legislative documents. Each process includes title, status, document type, dates, voting results, and detailed stages of the legislative procedure. Essential for tracking legislation through parliament, analyzing legislative progress, understanding voting patterns, and researching the complete lifecycle of parliamentary proposals from submission to final resolution.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 legislative activity. Each term has different legislative processes and priorities.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of processes to return (default: 50). Use higher values (e.g., '100', '200') for comprehensive legislative analysis.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through legislative processes.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Filter processes containing a specified string in the title (e.g., 'kodeks pracy', 'ustawa o', 'podatek').",
				},
				"document_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by document type (e.g., 'projekt ustawy', 'projekt uchwały'). Use to find specific types of legislative documents.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort processes by specified field. Add minus sign for descending order (e.g., '-changeDate' for most recently modified first, 'title' for alphabetical). Common fields: 'changeDate', 'title', 'processStartDate'.",
				},
			},
		},
	}, s.handleGetProcesses)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_processes_passed",
		Description: "Retrieve parliamentary legislative processes that have been successfully passed for a specific term. Returns information about completed legislation that went through all required stages and was adopted. Essential for studying successful legislative outcomes, analyzing passed legislation patterns, and understanding what types of bills successfully navigate the parliamentary process.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 passed legislation.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of passed processes to return (default: 50). Use higher values for comprehensive analysis of successful legislation.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through passed processes.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Filter passed processes containing a specified string in the title.",
				},
				"document_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by document type to see what types of legislation passed successfully.",
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"description": "Sort passed processes by specified field. Add minus sign for descending order (e.g., '-closureDate' for most recently passed first).",
				},
			},
		},
	}, s.handleGetProcessesPassed)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_process_details",
		Description: "Get detailed information about a specific legislative process including complete procedural history, voting records, committee work, amendments, and current status. Returns comprehensive process data with all stages, decisions, dates, and outcomes. Essential for detailed legislative analysis, understanding specific bill progress, tracking amendments and changes, and studying the complete parliamentary procedure for individual pieces of legislation.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 processes.",
				},
				"process_number": map[string]interface{}{
					"type":        "string",
					"description": "Process number (print number) to get details for (e.g., '1', '15', '100'). Get this from sejm_get_processes results.",
				},
			},
			Required: []string{"process_number"},
		},
	}, s.handleGetProcessDetails)
}

func (s *SejmServer) handleGetProcesses(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	if limit := request.GetString("limit", ""); limit != "" {
		params["limit"] = limit
	}
	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}
	if title := request.GetString("title", ""); title != "" {
		params["title"] = title
	}
	if documentType := request.GetString("document_type", ""); documentType != "" {
		params["documentType"] = documentType
	}
	if sortBy := request.GetString("sort_by", ""); sortBy != "" {
		params["sort_by"] = sortBy
	}

	s.logger.Info("sejm_get_processes called",
		slog.String("term", fmt.Sprintf("%d", term)),
		slog.Any("params", params))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/processes", term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch legislative processes: %v", err)), nil
	}

	var processes []sejm.ProcessHeader
	if err := json.Unmarshal(data, &processes); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse processes: %v", err)), nil
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Term: %d", term))
	summary = append(summary, fmt.Sprintf("Found %d legislative processes", len(processes)))

	// Add filter info
	if title := request.GetString("title", ""); title != "" {
		summary = append(summary, fmt.Sprintf("Title filter: '%s'", title))
	}
	if documentType := request.GetString("document_type", ""); documentType != "" {
		summary = append(summary, fmt.Sprintf("Document type: %s", documentType))
	}

	// Analyze data
	passedCount := 0
	billCount := 0
	resolutionCount := 0
	for _, process := range processes {
		if process.Passed != nil && *process.Passed {
			passedCount++
		}
		if process.DocumentTypeEnum != nil {
			switch *process.DocumentTypeEnum {
			case sejm.ProcessTypeBILL:
				billCount++
			case sejm.ProcessTypeDRAFTRESOLUTION:
				resolutionCount++
			}
		}
	}

	summary = append(summary, fmt.Sprintf("Passed: %d, Bills: %d, Resolutions: %d", passedCount, billCount, resolutionCount))

	var results []string
	if len(processes) == 0 {
		results = append(results, "No legislative processes found matching the criteria.")
		results = append(results, "")
		results = append(results, "Try adjusting your search parameters:")
		results = append(results, "• Remove filters to see all processes")
		results = append(results, "• Try different keywords in 'title' parameter")
		results = append(results, "• Use different document types")
	} else {
		// Show first 10 processes
		displayCount := 10
		if len(processes) < displayCount {
			displayCount = len(processes)
		}

		results = append(results, fmt.Sprintf("Showing first %d legislative processes:", displayCount))
		results = append(results, "")

		for i := 0; i < displayCount; i++ {
			process := processes[i]

			number := "Unknown"
			if process.Number != nil {
				number = *process.Number
			}

			title := "No title"
			if process.Title != nil {
				title = *process.Title
			}

			// Truncate long titles
			if len(title) > 120 {
				title = title[:117] + "..."
			}

			status := "In progress"
			if process.Passed != nil && *process.Passed {
				status = "PASSED"
			}

			docType := ""
			if process.DocumentType != nil {
				docType = fmt.Sprintf(" (%s)", *process.DocumentType)
			}

			startDate := ""
			if process.ProcessStartDate != nil {
				startDate = fmt.Sprintf(" - Started: %s", process.ProcessStartDate.Format("2006-01-02"))
			}

			results = append(results, fmt.Sprintf("Process #%s: %s [%s]%s%s", number, title, status, docType, startDate))
		}

		if len(processes) > displayCount {
			results = append(results, fmt.Sprintf("... and %d more processes", len(processes)-displayCount))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "Get process details: use sejm_get_process_details with process_number")
	nextActions = append(nextActions, "Filter by passed legislation: use sejm_get_processes_passed")
	nextActions = append(nextActions, "Search by title: use 'title' parameter with keywords")
	nextActions = append(nextActions, "Filter by type: use 'document_type' parameter")

	// Add pagination hints if we have results
	if len(processes) > 0 {
		if offset := request.GetString("offset", ""); offset == "" {
			nextActions = append(nextActions, "Next page: add offset='50' for more results")
		}
		if sortBy := request.GetString("sort_by", ""); sortBy == "" {
			nextActions = append(nextActions, "Sort by recent: add sort_by='-changeDate' for newest first")
		}
	}

	response := StandardResponse{
		Operation:   "Parliamentary Legislative Processes",
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Legislative processes track bills, resolutions, and other legislative documents through parliamentary procedure. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetProcessesPassed(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	if limit := request.GetString("limit", ""); limit != "" {
		params["limit"] = limit
	}
	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}
	if title := request.GetString("title", ""); title != "" {
		params["title"] = title
	}
	if documentType := request.GetString("document_type", ""); documentType != "" {
		params["documentType"] = documentType
	}
	if sortBy := request.GetString("sort_by", ""); sortBy != "" {
		params["sort_by"] = sortBy
	}

	s.logger.Info("sejm_get_processes_passed called",
		slog.String("term", fmt.Sprintf("%d", term)),
		slog.Any("params", params))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/processes/passed", term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch passed processes: %v", err)), nil
	}

	var processes []sejm.ProcessHeader
	if err := json.Unmarshal(data, &processes); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse passed processes: %v", err)), nil
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Term: %d", term))
	summary = append(summary, fmt.Sprintf("Found %d passed legislative processes", len(processes)))

	// Add filter info
	if title := request.GetString("title", ""); title != "" {
		summary = append(summary, fmt.Sprintf("Title filter: '%s'", title))
	}
	if documentType := request.GetString("document_type", ""); documentType != "" {
		summary = append(summary, fmt.Sprintf("Document type: %s", documentType))
	}

	var results []string
	if len(processes) == 0 {
		results = append(results, "No passed legislative processes found matching the criteria.")
		results = append(results, "")
		results = append(results, "This could mean:")
		results = append(results, "• No legislation matching your criteria has passed")
		results = append(results, "• Try broader search terms")
		results = append(results, "• Check different document types")
	} else {
		// Show first 10 passed processes
		displayCount := 10
		if len(processes) < displayCount {
			displayCount = len(processes)
		}

		results = append(results, fmt.Sprintf("Showing first %d passed processes:", displayCount))
		results = append(results, "")

		for i := 0; i < displayCount; i++ {
			process := processes[i]

			number := "Unknown"
			if process.Number != nil {
				number = *process.Number
			}

			title := "No title"
			if process.Title != nil {
				title = *process.Title
			}

			// Truncate long titles
			if len(title) > 120 {
				title = title[:117] + "..."
			}

			finalTitle := ""
			if process.TitleFinal != nil && *process.TitleFinal != "" {
				finalTitle = fmt.Sprintf(" → Final: %s", *process.TitleFinal)
			}

			closureDate := ""
			if process.ClosureDate != nil {
				closureDate = fmt.Sprintf(" - Passed: %s", process.ClosureDate.Format("2006-01-02"))
			}

			eli := ""
			if process.ELI != nil {
				eli = fmt.Sprintf(" [ELI: %s]", *process.ELI)
			}

			results = append(results, fmt.Sprintf("Process #%s: %s%s%s%s", number, title, finalTitle, closureDate, eli))
		}

		if len(processes) > displayCount {
			results = append(results, fmt.Sprintf("... and %d more passed processes", len(processes)-displayCount))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "Get process details: use sejm_get_process_details with process_number")
	nextActions = append(nextActions, "Search all processes: use sejm_get_processes for in-progress and passed")
	nextActions = append(nextActions, "Search by title: use 'title' parameter with keywords")

	// Add pagination hints if we have results
	if len(processes) > 0 {
		if offset := request.GetString("offset", ""); offset == "" {
			nextActions = append(nextActions, "Next page: add offset='50' for more results")
		}
		if sortBy := request.GetString("sort_by", ""); sortBy == "" {
			nextActions = append(nextActions, "Sort by recent: add sort_by='-closureDate' for newest passed first")
		}
	}

	response := StandardResponse{
		Operation:   "Passed Parliamentary Legislative Processes",
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("These are legislative processes that successfully completed all parliamentary stages and were adopted. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetProcessDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	processNumber := request.GetString("process_number", "")
	if processNumber == "" {
		return mcp.NewToolResultError("Process number is required. Please provide the process_number parameter. Get process numbers from sejm_get_processes results."), nil
	}

	s.logger.Info("sejm_get_process_details called",
		slog.String("term", fmt.Sprintf("%d", term)),
		slog.String("processNumber", processNumber))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/processes/%s", term, processNumber)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch process details: %v. Please verify process_number=%s exists in term %d.", err, processNumber, term)), nil
	}

	var process sejm.ProcessDetails
	if err := json.Unmarshal(data, &process); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse process details: %v", err)), nil
	}

	// Build comprehensive summary
	var summary []string
	summary = append(summary, fmt.Sprintf("Process #%s (Term %d)", processNumber, term))

	if process.Title != nil {
		summary = append(summary, fmt.Sprintf("Title: %s", *process.Title))
	}
	if process.TitleFinal != nil {
		summary = append(summary, fmt.Sprintf("Final title: %s", *process.TitleFinal))
	}

	status := "In progress"
	if process.Passed != nil && *process.Passed {
		status = "PASSED"
	}
	summary = append(summary, fmt.Sprintf("Status: %s", status))

	if process.DocumentType != nil {
		summary = append(summary, fmt.Sprintf("Document type: %s", *process.DocumentType))
	}

	var results []string

	// Basic information
	results = append(results, "📋 PROCESS OVERVIEW:")
	if process.ProcessStartDate != nil {
		results = append(results, fmt.Sprintf("• Started: %s", process.ProcessStartDate.Format("2006-01-02")))
	}
	if process.ClosureDate != nil {
		results = append(results, fmt.Sprintf("• Completed: %s", process.ClosureDate.Format("2006-01-02")))
	}
	if process.ELI != nil {
		results = append(results, fmt.Sprintf("• ELI: %s", *process.ELI))
	}
	if process.Address != nil {
		results = append(results, fmt.Sprintf("• Publication address: %s", *process.Address))
	}

	// Special flags
	specialFlags := []string{}
	if process.LegislativeCommittee != nil && *process.LegislativeCommittee {
		specialFlags = append(specialFlags, "Legislative Committee assigned")
	}
	if process.ShortenProcedure != nil && *process.ShortenProcedure {
		specialFlags = append(specialFlags, "Shortened procedure (Art. 51)")
	}
	if process.PrincipleOfSubsidiarity != nil && *process.PrincipleOfSubsidiarity {
		specialFlags = append(specialFlags, "Principle of subsidiarity issue")
	}
	if len(specialFlags) > 0 {
		results = append(results, "")
		results = append(results, "🏛️ SPECIAL PROCEDURES:")
		for _, flag := range specialFlags {
			results = append(results, fmt.Sprintf("• %s", flag))
		}
	}

	// Stages information
	if process.Stages != nil && len(*process.Stages) > 0 {
		results = append(results, "")
		results = append(results, "📈 LEGISLATIVE STAGES:")
		for i, stage := range *process.Stages {
			if i >= 8 { // Limit stages to prevent overwhelming output
				results = append(results, fmt.Sprintf("... and %d more stages", len(*process.Stages)-i))
				break
			}

			stageName := "Unknown stage"
			if stage.StageName != nil {
				stageName = *stage.StageName
			}

			date := ""
			if stage.Date != nil {
				date = fmt.Sprintf(" (%s)", stage.Date.Format("2006-01-02"))
			}

			stageType := ""
			if stage.StageType != nil {
				stageType = fmt.Sprintf(" [%s]", *stage.StageType)
			}

			results = append(results, fmt.Sprintf("%d. %s%s%s", i+1, stageName, date, stageType))
		}
	}

	// Related documents
	if process.PrintsConsideredJointly != nil && len(*process.PrintsConsideredJointly) > 0 {
		results = append(results, "")
		results = append(results, "📄 RELATED PRINTS:")
		for _, printNum := range *process.PrintsConsideredJointly {
			results = append(results, fmt.Sprintf("• Print #%s (considered jointly)", printNum))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "View all processes: use sejm_get_processes")
	nextActions = append(nextActions, "Find passed legislation: use sejm_get_processes_passed")
	if process.Stages != nil && len(*process.Stages) > 0 {
		nextActions = append(nextActions, "The stages field contains detailed procedural history")
	}
	if process.ELI != nil {
		nextActions = append(nextActions, fmt.Sprintf("Get legal text: use eli_get_act_text for %s", *process.ELI))
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Legislative Process #%s Details", processNumber),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Detailed legislative process information showing complete procedural history and current status. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) registerBilateralGroupsTools() {
	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_bilateral_groups",
		Description: "Retrieve parliamentary bilateral groups for a specific term. Bilateral groups are international parliamentary cooperation groups that facilitate diplomatic and political relationships between the Polish Parliament and other national parliaments. Returns information about group names, appointment dates, English names, and basic group details. Essential for understanding international parliamentary cooperation, analyzing diplomatic relationships, and researching Poland's international parliamentary engagement.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 bilateral groups. Each term may have different international cooperation arrangements.",
				},
				"limit": map[string]interface{}{
					"type":        "string",
					"description": "Maximum number of groups to return (default: 50). Use higher values for comprehensive analysis of international cooperation.",
				},
				"offset": map[string]interface{}{
					"type":        "string",
					"description": "Starting position within the collection of results (default: 0). Use with limit for pagination through bilateral groups.",
				},
			},
		},
	}, s.handleGetBilateralGroups)

	s.server.AddTool(mcp.Tool{
		Name:        "sejm_get_bilateral_group_details",
		Description: "Get detailed information about a specific bilateral group including complete membership list, member roles, appointment dates, and group description. Returns comprehensive group data with all current and former members, their parliamentary clubs, membership periods, and any special roles or positions within the group. Essential for analyzing specific international parliamentary relationships, understanding group composition, researching MP involvement in international cooperation, and studying detailed diplomatic parliamentary engagement.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"term": map[string]interface{}{
					"type":        "string",
					"description": "Parliamentary term number (1-10). Current term 10 covers 2019-2023 bilateral groups.",
				},
				"group_id": map[string]interface{}{
					"type":        "string",
					"description": "Bilateral group ID number. Get this from sejm_get_bilateral_groups results (e.g., '1', '5', '15').",
				},
			},
			Required: []string{"group_id"},
		},
	}, s.handleGetBilateralGroupDetails)
}

func (s *SejmServer) handleGetBilateralGroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	params := make(map[string]string)
	if limit := request.GetString("limit", ""); limit != "" {
		params["limit"] = limit
	}
	if offset := request.GetString("offset", ""); offset != "" {
		params["offset"] = offset
	}

	s.logger.Info("sejm_get_bilateral_groups called",
		slog.String("term", fmt.Sprintf("%d", term)),
		slog.Any("params", params))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/bilateralGroups", term)
	data, err := s.makeAPIRequest(ctx, endpoint, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch bilateral groups: %v", err)), nil
	}

	var groups []sejm.Group
	if err := json.Unmarshal(data, &groups); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse bilateral groups: %v", err)), nil
	}

	// Build response
	var summary []string
	summary = append(summary, fmt.Sprintf("Term: %d", term))
	summary = append(summary, fmt.Sprintf("Found %d bilateral groups", len(groups)))

	var results []string
	if len(groups) == 0 {
		results = append(results, "No bilateral groups found for this term.")
		results = append(results, "")
		results = append(results, "This could mean:")
		results = append(results, "• No international parliamentary cooperation groups established")
		results = append(results, "• Term period may not have bilateral group data available")
		results = append(results, "• Try a different term number")
	} else {
		// Show bilateral groups
		displayCount := 15
		if len(groups) < displayCount {
			displayCount = len(groups)
		}

		results = append(results, fmt.Sprintf("Showing first %d bilateral groups:", displayCount))
		results = append(results, "")

		for i := 0; i < displayCount; i++ {
			group := groups[i]

			id := "Unknown"
			if group.Id != nil {
				id = fmt.Sprintf("%d", *group.Id)
			}

			name := "No name"
			if group.Name != nil {
				name = *group.Name
			}

			// Truncate long names
			if len(name) > 80 {
				name = name[:77] + "..."
			}

			engName := ""
			if group.EngName != nil && *group.EngName != "" {
				engName = fmt.Sprintf(" (%s)", *group.EngName)
			}

			appointmentDate := ""
			if group.AppointmentDate != nil {
				appointmentDate = fmt.Sprintf(" - Appointed: %s", group.AppointmentDate.Format("2006-01-02"))
			}

			results = append(results, fmt.Sprintf("Group #%s: %s%s%s", id, name, engName, appointmentDate))
		}

		if len(groups) > displayCount {
			results = append(results, fmt.Sprintf("... and %d more groups", len(groups)-displayCount))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "Get group details: use sejm_get_bilateral_group_details with group_id")
	nextActions = append(nextActions, "View all MPs: use sejm_get_mps for member identification")

	// Add pagination hints if we have results
	if len(groups) > 0 {
		if offset := request.GetString("offset", ""); offset == "" {
			nextActions = append(nextActions, "Next page: add offset='50' for more results")
		}
	}

	response := StandardResponse{
		Operation:   "Parliamentary Bilateral Groups",
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Bilateral groups facilitate international parliamentary cooperation and diplomatic relationships. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetBilateralGroupDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	term, err := s.validateTerm(request.GetString("term", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parliamentary term: %v. Please use term numbers 1-10.", err)), nil
	}

	groupID := request.GetString("group_id", "")
	if groupID == "" {
		return mcp.NewToolResultError("Group ID is required. Please provide the group_id parameter. Get group IDs from sejm_get_bilateral_groups results."), nil
	}

	s.logger.Info("sejm_get_bilateral_group_details called",
		slog.String("term", fmt.Sprintf("%d", term)),
		slog.String("groupID", groupID))

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%d/bilateralGroups/%s", term, groupID)
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch bilateral group details: %v. Please verify group_id=%s exists in term %d.", err, groupID, term)), nil
	}

	var groupDetails sejm.GroupDetails
	if err := json.Unmarshal(data, &groupDetails); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse bilateral group details: %v", err)), nil
	}

	// Build comprehensive summary
	var summary []string
	summary = append(summary, fmt.Sprintf("Group #%s (Term %d)", groupID, term))

	if groupDetails.Name != nil {
		summary = append(summary, fmt.Sprintf("Name: %s", *groupDetails.Name))
	}
	if groupDetails.EngName != nil {
		summary = append(summary, fmt.Sprintf("English name: %s", *groupDetails.EngName))
	}

	memberCount := 0
	if groupDetails.Members != nil {
		memberCount = len(*groupDetails.Members)
	}
	summary = append(summary, fmt.Sprintf("Members: %d", memberCount))

	var results []string

	// Basic information
	results = append(results, "🌍 GROUP OVERVIEW:")
	if groupDetails.AppointmentDate != nil {
		results = append(results, fmt.Sprintf("• Appointed: %s", groupDetails.AppointmentDate.Format("2006-01-02")))
	}
	if groupDetails.Remarks != nil && *groupDetails.Remarks != "" {
		results = append(results, fmt.Sprintf("• Remarks: %s", *groupDetails.Remarks))
	}

	// Members information
	if groupDetails.Members != nil && len(*groupDetails.Members) > 0 {
		results = append(results, "")
		results = append(results, "👥 GROUP MEMBERS:")

		// Count active vs former members
		activeMembers := 0
		formerMembers := 0
		for _, member := range *groupDetails.Members {
			if member.MembershipEnd == nil {
				activeMembers++
			} else {
				formerMembers++
			}
		}

		results = append(results, fmt.Sprintf("• Active members: %d", activeMembers))
		results = append(results, fmt.Sprintf("• Former members: %d", formerMembers))
		results = append(results, "")

		// Show first 15 members to avoid overwhelming output
		displayCount := 15
		if len(*groupDetails.Members) < displayCount {
			displayCount = len(*groupDetails.Members)
		}

		results = append(results, fmt.Sprintf("Member details (showing first %d):", displayCount))
		for i := 0; i < displayCount; i++ {
			member := (*groupDetails.Members)[i]

			name := "Unknown"
			if member.Name != nil {
				name = *member.Name
			}

			club := ""
			if member.Club != nil {
				club = fmt.Sprintf(" (%s)", *member.Club)
			}

			memberType := ""
			if member.Type != nil {
				memberType = fmt.Sprintf(" [%s]", string(*member.Type))
			}

			status := ""
			if member.MembershipEnd == nil {
				status = " - ACTIVE"
			} else {
				status = fmt.Sprintf(" - ended %s", member.MembershipEnd.Format("2006-01-02"))
			}

			senatorNote := ""
			if member.Senator != nil && *member.Senator {
				senatorNote = " (Senator)"
			}

			results = append(results, fmt.Sprintf("• %s%s%s%s%s", name, club, senatorNote, memberType, status))
		}

		if len(*groupDetails.Members) > displayCount {
			results = append(results, fmt.Sprintf("... and %d more members", len(*groupDetails.Members)-displayCount))
		}
	}

	// Build next actions
	var nextActions []string
	nextActions = append(nextActions, "View all bilateral groups: use sejm_get_bilateral_groups")
	nextActions = append(nextActions, "Get MP details: use sejm_get_mp_details with specific MP IDs")
	if groupDetails.Members != nil && len(*groupDetails.Members) > 0 {
		nextActions = append(nextActions, "The members field contains complete membership history with dates")
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Bilateral Group #%s Details", groupID),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Detailed bilateral group information showing complete membership and cooperation details. Data retrieved from term %d on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetInterpellationBody(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_interpellation_body called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	num := request.GetString("num", "")

	if term == "" || num == "" {
		return mcp.NewToolResultError("Both 'term' and 'num' parameters are required. Get these from sejm_get_interpellations results."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/interpellations/%s/body", term, num)

	// Use text request for HTML content
	data, err := s.makeTextRequest(ctx, endpoint, "html")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve interpellation body: %v", err)), nil
	}

	response := StandardResponse{
		Operation: fmt.Sprintf("Interpellation #%s Body (Term %s)", num, term),
		Status:    "Retrieved Successfully",
		Summary:   []string{fmt.Sprintf("Full HTML content of interpellation #%s from parliamentary term %s", num, term)},
		Data:      []string{string(data)},
		NextActions: []string{
			fmt.Sprintf("Get replies: sejm_get_interpellation_reply_body with term='%s' and num='%s'", term, num),
			fmt.Sprintf("View interpellation list: sejm_get_interpellations with term='%s'", term),
		},
		Note: fmt.Sprintf("Interpellation body content retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetInterpellationReplyBody(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_interpellation_reply_body called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	num := request.GetString("num", "")
	key := request.GetString("key", "")

	if term == "" || num == "" || key == "" {
		return mcp.NewToolResultError("All parameters 'term', 'num', and 'key' are required. Get these from sejm_get_interpellations results."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/interpellations/%s/reply/%s/body", term, num, key)

	// Use text request for HTML content
	data, err := s.makeTextRequest(ctx, endpoint, "html")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve interpellation reply body: %v", err)), nil
	}

	response := StandardResponse{
		Operation: fmt.Sprintf("Interpellation #%s Reply Body (Term %s, Key %s)", num, term, key),
		Status:    "Retrieved Successfully",
		Summary:   []string{fmt.Sprintf("Full HTML content of government reply to interpellation #%s from parliamentary term %s", num, term)},
		Data:      []string{string(data)},
		NextActions: []string{
			fmt.Sprintf("Get original question: sejm_get_interpellation_body with term='%s' and num='%s'", term, num),
			fmt.Sprintf("View interpellation list: sejm_get_interpellations with term='%s'", term),
		},
		Note: fmt.Sprintf("Government reply content retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetInterpellationAttachment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_interpellation_attachment called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	key := request.GetString("key", "")
	fileName := request.GetString("file_name", "")

	if term == "" || key == "" || fileName == "" {
		return mcp.NewToolResultError("All parameters 'term', 'key', and 'file_name' are required. Get these from interpellation details."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/interpellations/attachment/%s/%s", term, key, fileName)

	// Use binary request for attachment files
	data, err := s.makeAPIRequestWithHeaders(ctx, endpoint, nil, map[string]string{"Accept": "*/*"})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve interpellation attachment: %v", err)), nil
	}

	// For binary files, we should provide metadata instead of raw content
	response := StandardResponse{
		Operation: fmt.Sprintf("Interpellation Attachment: %s (Term %s)", fileName, term),
		Status:    "Retrieved Successfully",
		Summary: []string{
			fmt.Sprintf("Downloaded attachment file '%s' from interpellation (key: %s)", fileName, key),
			fmt.Sprintf("File size: %d bytes", len(data)),
		},
		Data: []string{fmt.Sprintf("Binary file content available (%d bytes). File type can be determined from extension: %s", len(data), fileName)},
		NextActions: []string{
			fmt.Sprintf("Get interpellation details: sejm_get_interpellations with term='%s'", term),
			"Process the binary content based on file type (PDF, DOC, image, etc.)",
		},
		Note: fmt.Sprintf("Attachment file downloaded from term %s on %s. Binary content available for further processing.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetPrintDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_print_details called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	num := request.GetString("num", "")

	if term == "" || num == "" {
		return mcp.NewToolResultError("Both 'term' and 'num' parameters are required. Get these from sejm_get_prints results."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/prints/%s", term, num)

	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve print details: %v", err)), nil
	}

	var printData sejm.Print
	if err := json.Unmarshal(data, &printData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse print data: %v", err)), nil
	}

	// Build summary information
	var summary []string
	var results []string
	var nextActions []string

	if printData.Title != nil {
		summary = append(summary, fmt.Sprintf("Title: %s", *printData.Title))
	}

	if printData.Number != nil {
		summary = append(summary, fmt.Sprintf("Print Number: %s", *printData.Number))
	}

	if printData.DeliveryDate != nil {
		summary = append(summary, fmt.Sprintf("Delivery Date: %s", printData.DeliveryDate.Format("2006-01-02")))
	}

	// Add complete details
	printJSON, _ := json.MarshalIndent(printData, "", "  ")
	results = append(results, string(printJSON))

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("View all prints: sejm_get_prints with term='%s'", term))

	if printData.Attachments != nil && len(*printData.Attachments) > 0 {
		nextActions = append(nextActions, fmt.Sprintf("Download attachments: sejm_get_print_attachment with term='%s' and num='%s'", term, num))
		summary = append(summary, fmt.Sprintf("Attachments available: %d files", len(*printData.Attachments)))
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Print #%s Details (Term %s)", num, term),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Print details retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetPrintAttachment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_print_attachment called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	num := request.GetString("num", "")
	attachName := request.GetString("attach_name", "")

	if term == "" || num == "" || attachName == "" {
		return mcp.NewToolResultError("All parameters 'term', 'num', and 'attach_name' are required. Get these from print details."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/prints/%s/%s", term, num, attachName)

	// Use binary request for attachment files
	data, err := s.makeAPIRequestWithHeaders(ctx, endpoint, nil, map[string]string{"Accept": "*/*"})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve print attachment: %v", err)), nil
	}

	// For binary files, we should provide metadata instead of raw content
	response := StandardResponse{
		Operation: fmt.Sprintf("Print Attachment: %s (Term %s, Print #%s)", attachName, term, num),
		Status:    "Retrieved Successfully",
		Summary: []string{
			fmt.Sprintf("Downloaded attachment file '%s' from print #%s", attachName, num),
			fmt.Sprintf("File size: %d bytes", len(data)),
		},
		Data: []string{fmt.Sprintf("Binary file content available (%d bytes). File type can be determined from extension: %s", len(data), attachName)},
		NextActions: []string{
			fmt.Sprintf("Get print details: sejm_get_print_details with term='%s' and num='%s'", term, num),
			fmt.Sprintf("View all prints: sejm_get_prints with term='%s'", term),
			"Process the binary content based on file type (PDF, DOC, image, etc.)",
		},
		Note: fmt.Sprintf("Attachment file downloaded from term %s on %s. Binary content available for further processing.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetClubDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_club_details called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	clubID := request.GetString("club_id", "")

	if term == "" || clubID == "" {
		return mcp.NewToolResultError("Both 'term' and 'club_id' parameters are required. Get these from sejm_get_clubs results."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/clubs/%s", term, clubID)

	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve club details: %v", err)), nil
	}

	var club sejm.Club
	if err := json.Unmarshal(data, &club); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse club data: %v", err)), nil
	}

	// Build summary information
	var summary []string
	var results []string
	var nextActions []string

	if club.Name != nil {
		summary = append(summary, fmt.Sprintf("Club Name: %s", *club.Name))
	}

	if club.Id != nil {
		summary = append(summary, fmt.Sprintf("Club ID: %s", *club.Id))
	}

	if club.MembersCount != nil {
		summary = append(summary, fmt.Sprintf("Members: %d", *club.MembersCount))
	}

	// Add complete details
	clubJSON, _ := json.MarshalIndent(club, "", "  ")
	results = append(results, string(clubJSON))

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("View all clubs: sejm_get_clubs with term='%s'", term))
	nextActions = append(nextActions, fmt.Sprintf("View club MPs: sejm_get_mps with term='%s' (filter by club)", term))

	response := StandardResponse{
		Operation:   fmt.Sprintf("Club Details: %s (Term %s)", clubID, term),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Club details retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetCommitteeDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_committee_details called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")
	committeeCode := request.GetString("committee_code", "")

	if term == "" || committeeCode == "" {
		return mcp.NewToolResultError("Both 'term' and 'committee_code' parameters are required. Get these from sejm_get_committees results."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/committees/%s", term, committeeCode)

	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve committee details: %v", err)), nil
	}

	var committee sejm.Committee
	if err := json.Unmarshal(data, &committee); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse committee data: %v", err)), nil
	}

	// Build summary information
	var summary []string
	var results []string
	var nextActions []string

	if committee.Name != nil {
		summary = append(summary, fmt.Sprintf("Committee Name: %s", *committee.Name))
	}

	if committee.Code != nil {
		summary = append(summary, fmt.Sprintf("Committee Code: %s", *committee.Code))
	}

	if committee.Type != nil {
		summary = append(summary, fmt.Sprintf("Committee Type: %s", string(*committee.Type)))
	}

	// Add member count if available
	memberCount := 0
	if committee.Members != nil {
		memberCount = len(*committee.Members)
		summary = append(summary, fmt.Sprintf("Members: %d", memberCount))
	}

	// Add complete details
	committeeJSON, _ := json.MarshalIndent(committee, "", "  ")
	results = append(results, string(committeeJSON))

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("View all committees: sejm_get_committees with term='%s'", term))
	nextActions = append(nextActions, fmt.Sprintf("View committee sittings: sejm_get_committee_sittings with term='%s' and committee_code='%s'", term, committeeCode))

	response := StandardResponse{
		Operation:   fmt.Sprintf("Committee Details: %s (Term %s)", committeeCode, term),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Committee details retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}

func (s *SejmServer) handleGetCurrentProceeding(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Info("sejm_get_current_proceeding called", slog.Any("arguments", request.Params.Arguments))

	term := request.GetString("term", "")

	if term == "" {
		return mcp.NewToolResultError("'term' parameter is required."), nil
	}

	endpoint := fmt.Sprintf("https://api.sejm.gov.pl/sejm/term%s/proceedings/current", term)

	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve current proceeding: %v", err)), nil
	}

	var proceeding sejm.Proceeding
	if err := json.Unmarshal(data, &proceeding); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse proceeding data: %v", err)), nil
	}

	// Build summary information
	var summary []string
	var results []string
	var nextActions []string

	if proceeding.Number != nil {
		summary = append(summary, fmt.Sprintf("Proceeding Number: %d", *proceeding.Number))
	}

	if proceeding.Dates != nil && len(*proceeding.Dates) > 0 {
		dates := *proceeding.Dates
		if len(dates) > 0 {
			summary = append(summary, fmt.Sprintf("Date: %s", dates[0].Format("2006-01-02")))
		}
	}

	if proceeding.Current != nil {
		status := "Inactive"
		if *proceeding.Current {
			status = "Currently Active"
		}
		summary = append(summary, fmt.Sprintf("Status: %s", status))
	}

	// Add essential proceeding details (compact format to avoid large responses)
	if proceeding.Title != nil {
		results = append(results, fmt.Sprintf("Title: %s", *proceeding.Title))
	}

	if proceeding.Dates != nil && len(*proceeding.Dates) > 0 {
		dates := *proceeding.Dates
		if len(dates) == 1 {
			results = append(results, fmt.Sprintf("Date: %s", dates[0].Format("2006-01-02")))
		} else {
			dateStrs := make([]string, len(dates))
			for i, date := range dates {
				dateStrs[i] = date.Format("2006-01-02")
			}
			results = append(results, fmt.Sprintf("Dates: %s (%d days)", strings.Join(dateStrs, ", "), len(dates)))
		}
	}

	if proceeding.Current != nil {
		if *proceeding.Current {
			results = append(results, "Status: Currently active proceeding")
		} else {
			results = append(results, "Status: Proceeding completed")
		}
	}

	// Show agenda if available (but not full details to keep response small)
	if proceeding.Agenda != nil && *proceeding.Agenda != "" {
		agenda := *proceeding.Agenda
		// If agenda is very long, truncate it
		if len(agenda) > 200 {
			agenda = agenda[:200] + "..."
		}
		results = append(results, fmt.Sprintf("Agenda: %s", agenda))
		results = append(results, "💡 Use sejm_get_transcripts to view detailed proceedings and full agenda")
	} else {
		results = append(results, "Agenda: No agenda information available")
	}

	// Suggest next actions
	nextActions = append(nextActions, fmt.Sprintf("View all proceedings: sejm_get_proceedings with term='%s'", term))
	if proceeding.Number != nil {
		nextActions = append(nextActions, fmt.Sprintf("View transcripts: sejm_get_transcripts with term='%s' and proceeding_id='%d'", term, *proceeding.Number))
	}

	response := StandardResponse{
		Operation:   fmt.Sprintf("Current Proceeding (Term %s)", term),
		Status:      "Retrieved Successfully",
		Summary:     summary,
		Data:        results,
		NextActions: nextActions,
		Note:        fmt.Sprintf("Current proceeding information retrieved from term %s on %s.", term, time.Now().Format("2006-01-02 15:04:05 MST")),
	}

	return mcp.NewToolResultText(response.Format()), nil
}
