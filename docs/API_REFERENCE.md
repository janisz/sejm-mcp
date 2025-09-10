# API Reference

Complete reference for all MCP tools provided by the Sejm MCP Server.

## Overview

The Sejm MCP Server provides 10 tools across two main API categories:
- **5 Sejm API tools**: Parliamentary data (MPs, committees, votings, interpellations)
- **5 ELI API tools**: Legal document database access

All tools return JSON data that can be processed by AI assistants or applications.

## Error Handling

All tools use consistent error handling:
- **Invalid parameters**: Returns error message with parameter requirements
- **API failures**: Returns error with HTTP status and description
- **Network timeouts**: 30-second timeout with descriptive error message
- **Invalid term**: Term must be between 1-10 for Sejm APIs

## Rate Limiting

The underlying Polish government APIs may implement rate limiting. The server handles this gracefully by:
- Using appropriate HTTP timeouts
- Providing informative error messages
- Suggesting retry strategies in documentation

---

# Sejm API Tools

## sejm_get_mps

**Description**: Retrieve a list of Members of Parliament for a specific parliamentary term.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `term` | string | No | "10" | Parliamentary term number (1-10) |

### Example Request
```json
{
  "tool": "sejm_get_mps",
  "arguments": {
    "term": "10"
  }
}
```

### Response Format
Returns an array of MP objects with the following structure:

```json
[
  {
    "id": 123,
    "firstName": "Jan",
    "lastName": "Kowalski",
    "firstLastName": "Jan Kowalski",
    "lastFirstName": "Kowalski Jan",
    "club": "Klub Parlamentarny",
    "districtNum": 1,
    "districtName": "District Name",
    "voivodeship": "Voivodeship",
    "active": true,
    "email": "j.kowalski@sejm.gov.pl",
    "numberOfVotes": 15420,
    "profession": "lawyer",
    "educationLevel": "higher",
    "birthDate": "1980-01-15",
    "birthLocation": "Warsaw"
  }
]
```

### Use Cases
- Generate contact lists for parliamentary research
- Analyze MP demographics and backgrounds
- Track active vs inactive MPs
- Build political party membership overviews

---

## sejm_get_mp_details

**Description**: Get detailed information about a specific Member of Parliament, including biography and statistics.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `term` | string | No | "10" | Parliamentary term number (1-10) |
| `mp_id` | string | Yes | - | MP identification number |

### Example Request
```json
{
  "tool": "sejm_get_mp_details",
  "arguments": {
    "term": "10",
    "mp_id": "123"
  }
}
```

### Response Format
Returns detailed MP object with additional fields:

```json
{
  "id": 123,
  "firstName": "Jan",
  "lastName": "Kowalski",
  "club": "Klub Parlamentarny",
  "active": true,
  "email": "j.kowalski@sejm.gov.pl",
  "profession": "lawyer",
  "educationLevel": "higher",
  "birthDate": "1980-01-15",
  "birthLocation": "Warsaw",
  "districtNum": 1,
  "districtName": "Warsaw I",
  "voivodeship": "mazowieckie",
  "numberOfVotes": 15420,
  "genitiveName": "Jana Kowalskiego",
  "accusativeName": "Jana Kowalskiego",
  "secondName": "Aleksander",
  "inactiveCause": null,
  "waiverDesc": null
}
```

### Use Cases
- Create detailed MP profiles for analysis
- Research MP backgrounds and qualifications
- Generate biographical summaries
- Track MP activity and engagement

---

## sejm_get_committees

**Description**: List all parliamentary committees for a specific term, including membership and contact information.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `term` | string | No | "10" | Parliamentary term number (1-10) |

### Example Request
```json
{
  "tool": "sejm_get_committees",
  "arguments": {
    "term": "10"
  }
}
```

### Response Format
Returns array of committee objects:

```json
[
  {
    "code": "ZKS",
    "name": "Committee on Health",
    "nameGenitive": "Komisji Zdrowia",
    "type": "STANDING",
    "scope": "Healthcare policy and legislation",
    "phone": "+48 22 694 1234",
    "appointmentDate": "2019-11-12",
    "compositionDate": "2019-11-15",
    "members": [
      {
        "id": 123,
        "lastFirstName": "Kowalski Jan",
        "club": "Club Name",
        "function": "chairman",
        "mandateExpired": null
      }
    ],
    "subCommittees": ["Subcommittee on Mental Health"]
  }
]
```

### Committee Types
- `STANDING`: Permanent parliamentary committees
- `EXTRAORDINARY`: Special purpose committees
- `INVESTIGATIVE`: Parliamentary investigation committees

### Use Cases
- Map parliamentary committee structure
- Track committee membership changes
- Analyze workload distribution across committees
- Find expert MPs in specific policy areas

---

## sejm_search_votings

**Description**: Search parliamentary voting records with filtering options for analysis.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `term` | string | No | "10" | Parliamentary term number (1-10) |
| `sitting` | string | No | - | Specific sitting number |
| `limit` | string | No | "50" | Maximum number of results |

### Example Request
```json
{
  "tool": "sejm_search_votings",
  "arguments": {
    "term": "10",
    "sitting": "1",
    "limit": "25"
  }
}
```

### Response Format
Returns array of voting objects:

```json
[
  {
    "votingNumber": 1,
    "date": "2019-11-12T14:30:00Z",
    "sitting": 1,
    "sittingDay": 1,
    "title": "Budget Act Vote",
    "topic": "Third reading of budget bill",
    "description": "Final vote on 2020 state budget",
    "kind": "ELECTRONIC",
    "majorityType": "SIMPLE_MAJORITY",
    "majorityVotes": 231,
    "yes": 245,
    "no": 180,
    "abstain": 25,
    "notParticipating": 10,
    "totalVoted": 450,
    "term": 10
  }
]
```

### Voting Kinds
- `ELECTRONIC`: Electronic voting system
- `TRADITIONAL`: Traditional (show of hands)
- `ON_LIST`: Voting on candidate lists

### Majority Types
- `SIMPLE_MAJORITY`: More than 50% of votes cast
- `ABSOLUTE_MAJORITY`: More than 50% of all MPs
- `STATUTORY_MAJORITY`: Special constitutional majority
- Various supermajority types (2/3, 3/5)

### Use Cases
- Analyze voting patterns and trends
- Track MP attendance and participation
- Study legislative success rates
- Research coalition behavior

---

## sejm_get_interpellations

**Description**: Retrieve parliamentary interpellations (formal questions submitted by MPs to government ministers).

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `term` | string | No | "10" | Parliamentary term number (1-10) |
| `limit` | string | No | "50" | Maximum number of results |

### Example Request
```json
{
  "tool": "sejm_get_interpellations",
  "arguments": {
    "term": "10",
    "limit": "20"
  }
}
```

### Response Format
Returns array of interpellation objects:

```json
[
  {
    "num": 1234,
    "title": "Healthcare funding in rural areas",
    "from": ["123", "456"],
    "to": ["Minister of Health"],
    "receiptDate": "2020-01-15",
    "sentDate": "2020-01-20",
    "lastModified": "2020-02-01T10:30:00Z",
    "term": 10,
    "answerDelayedDays": 5,
    "recipientDetails": [
      {
        "name": "Ministry of Health",
        "sent": "2020-01-20",
        "answerDelayedDays": 5
      }
    ],
    "replies": [
      {
        "key": "reply-001",
        "from": "Minister of Health",
        "receiptDate": "2020-02-01",
        "lastModified": "2020-02-01T10:30:00Z",
        "prolongation": false,
        "onlyAttachment": false
      }
    ]
  }
]
```

### Use Cases
- Monitor government accountability
- Track ministerial response times
- Analyze MP inquiry patterns
- Research policy concerns and priorities

---

# ELI API Tools

## eli_search_acts

**Description**: Search the Polish legal acts database with advanced filtering capabilities.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `title` | string | No | - | Search keywords in act titles |
| `publisher` | string | No | - | Publisher code (e.g., "DU") |
| `year` | string | No | - | Publication year |
| `type` | string | No | - | Document type |
| `limit` | string | No | "50" | Maximum number of results |

### Example Request
```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "konstytucja",
    "publisher": "DU",
    "year": "1997",
    "limit": "10"
  }
}
```

### Response Format
Returns search results object:

```json
{
  "items": [
    {
      "eli": "http://eli.gov.pl/eli/DU/1997/78/pol/t",
      "title": "Constitution of the Republic of Poland",
      "publisher": "DU",
      "year": 1997,
      "position": 78,
      "actDate": "1997-04-02",
      "publishDate": "1997-04-16",
      "status": "obowiązujący",
      "type": "konstytucja"
    }
  ],
  "count": 1
}
```

### Common Publishers
- `DU`: Journal of Laws (Dziennik Ustaw)
- `MP`: Monitor Polski (Polish Monitor)
- `DzUrz`: Official gazettes of ministries

### Use Cases
- Legal research and citation finding
- Legislative history tracking
- Compliance and regulatory analysis
- Academic legal research

---

## eli_get_act_details

**Description**: Retrieve comprehensive metadata for a specific legal act using its publication identifiers.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `publisher` | string | Yes | - | Publisher code (e.g., "DU") |
| `year` | string | Yes | - | Publication year |
| `position` | string | Yes | - | Position number in journal |

### Example Request
```json
{
  "tool": "eli_get_act_details",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78"
  }
}
```

### Response Format
Returns detailed act metadata:

```json
{
  "eli": "http://eli.gov.pl/eli/DU/1997/78/pol/t",
  "title": "Konstytucja Rzeczypospolitej Polskiej",
  "publisher": "DU",
  "year": 1997,
  "position": 78,
  "volume": 1,
  "actDate": "1997-04-02",
  "publishDate": "1997-04-16",
  "effectiveDate": "1997-10-17",
  "status": "obowiązujący",
  "type": "konstytucja",
  "institution": "Zgromadzenie Narodowe",
  "keywords": ["konstytucja", "prawo podstawowe", "ustrój"],
  "lastModified": "2020-01-01T00:00:00Z",
  "textFormats": ["HTML", "PDF"],
  "relatedActs": 145
}
```

### Status Values
- `obowiązujący`: Currently in force
- `uchylony`: Repealed
- `wygasł`: Expired
- `nieobowiązujący`: Not in force

### Use Cases
- Legal document verification
- Citation formatting and validation
- Regulatory impact analysis
- Legal database management

---

## eli_get_act_text

**Description**: Download the full text of a legal act in HTML or PDF format for analysis or display.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `publisher` | string | Yes | - | Publisher code |
| `year` | string | Yes | - | Publication year |
| `position` | string | Yes | - | Position number |
| `format` | string | No | "html" | Format: "html" or "pdf" |

### Example Request
```json
{
  "tool": "eli_get_act_text",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78",
    "format": "html"
  }
}
```

### Response Format

**HTML Format**: Returns formatted legal text with structure:
```html
<html>
<head>
  <title>Konstytucja Rzeczypospolitej Polskiej</title>
</head>
<body>
  <div class="act-header">
    <h1>KONSTYTUCJA RZECZYPOSPOLITEJ POLSKIEJ</h1>
    <p class="date">z dnia 2 kwietnia 1997 r.</p>
  </div>
  <div class="act-content">
    <div class="chapter">
      <h2>ROZDZIAŁ I<br/>RZECZPOSPOLITA</h2>
      <div class="article">
        <h3>Art. 1.</h3>
        <p>Rzeczpospolita Polska jest dobrem wspólnym wszystkich obywateli.</p>
      </div>
    </div>
  </div>
</body>
</html>
```

**PDF Format**: Returns binary PDF content with message:
```
PDF content for act DU/1997/78 retrieved (245760 bytes)
```

### Use Cases
- Legal document analysis and AI processing
- Creating legal citations and references
- Building legal knowledge bases
- Educational and research applications

---

## eli_get_act_references

**Description**: Explore legal relationships between acts, including citations, amendments, and dependencies.

### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `publisher` | string | Yes | - | Publisher code |
| `year` | string | Yes | - | Publication year |
| `position` | string | Yes | - | Position number |

### Example Request
```json
{
  "tool": "eli_get_act_references",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78"
  }
}
```

### Response Format
Returns array of reference objects:

```json
[
  {
    "type": "zmiana",
    "direction": "outgoing",
    "targetEli": "http://eli.gov.pl/eli/DU/2006/143/pol",
    "targetTitle": "Ustawa o zmianie Konstytucji",
    "targetPublisher": "DU",
    "targetYear": 2006,
    "targetPosition": 143,
    "description": "Zmiana art. 55 Konstytucji",
    "date": "2006-09-08"
  },
  {
    "type": "powołanie",
    "direction": "incoming",
    "sourceEli": "http://eli.gov.pl/eli/DU/1998/106/pol",
    "sourceTitle": "Ustawa o samorządzie gminnym",
    "description": "Powołanie na art. 16 Konstytucji"
  }
]
```

### Reference Types
- `zmiana`: Amendment or modification
- `powołanie`: Citation or reference
- `uchylenie`: Repeal
- `podstawa`: Legal basis
- `wykonanie`: Implementation act

### Directions
- `outgoing`: This act references another
- `incoming`: Another act references this one
- `bidirectional`: Mutual reference

### Use Cases
- Legal dependency mapping
- Amendment tracking and history
- Citation network analysis
- Regulatory impact assessment

---

## eli_get_publishers

**Description**: List all available legal document publishers in the ELI database.

### Parameters
None required.

### Example Request
```json
{
  "tool": "eli_get_publishers",
  "arguments": {}
}
```

### Response Format
Returns array of publisher objects:

```json
[
  {
    "code": "DU",
    "name": "Dziennik Ustaw Rzeczypospolitej Polskiej",
    "nameEn": "Journal of Laws of the Republic of Poland",
    "description": "Główny dziennik urzędowy publikacji aktów prawnych",
    "website": "https://dziennikustaw.gov.pl",
    "actCount": 45678,
    "yearRange": {
      "from": 1918,
      "to": 2024
    },
    "active": true
  },
  {
    "code": "MP",
    "name": "Monitor Polski",
    "nameEn": "Polish Monitor",
    "description": "Dziennik urzędowy Rzeczypospolitej Polskiej",
    "actCount": 23456,
    "active": true
  }
]
```

### Major Publishers
- **DU**: Primary journal for laws and regulations
- **MP**: Official gazette for administrative acts
- **DzUrz**: Ministry-specific gazettes
- **Regional**: Voivodeship and local government publications

### Use Cases
- Legal database exploration and setup
- Publisher-specific research projects
- Understanding Polish legal publication system
- Building comprehensive legal search tools

---

# Data Models

## Common Types

### Date Formats
- **ISO 8601**: `2020-01-15T10:30:00Z` (with timezone)
- **Date only**: `2020-01-15` (YYYY-MM-DD)

### Identifiers
- **MP ID**: Integer, unique within term
- **ELI**: Standard European format: `http://eli.gov.pl/eli/{publisher}/{year}/{position}/pol`
- **Committee Code**: 2-4 letter abbreviation
- **Voting Number**: Sequential integer within sitting

### Pagination
Many endpoints support pagination parameters:
- `limit`: Maximum results (default varies by endpoint)
- `offset`: Starting position (some endpoints)
- `sort_by`: Field name, prefix with `-` for descending

### Error Responses
```json
{
  "error": "Invalid term: must be between 1 and 10",
  "code": "INVALID_PARAMETER",
  "parameter": "term"
}
```

---

# Integration Examples

## Legal Research Assistant
```python
# Search for constitutional amendments
constitutional_acts = eli_search_acts(
    title="konstytucja",
    publisher="DU",
    limit="100"
)

# Get full text for analysis
for act in constitutional_acts['items']:
    text = eli_get_act_text(
        publisher=act['publisher'],
        year=str(act['year']),
        position=str(act['position']),
        format="html"
    )
    # Process with NLP tools
```

## Parliamentary Analysis
```python
# Get current MPs and their voting records
mps = sejm_get_mps(term="10")
for mp in mps:
    if mp['active']:
        details = sejm_get_mp_details(
            term="10",
            mp_id=str(mp['id'])
        )
        # Analyze MP activity and specializations
```

## Government Accountability Tracking
```python
# Monitor interpellation response times
interpellations = sejm_get_interpellations(
    term="10",
    limit="200"
)

delayed_responses = [
    i for i in interpellations
    if i.get('answerDelayedDays', 0) > 30
]
# Generate accountability reports
```