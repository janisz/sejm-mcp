# Usage Examples

This document provides practical examples of using the Sejm MCP Server for various research and analysis tasks.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Parliamentary Research](#parliamentary-research)
3. [Legal Document Analysis](#legal-document-analysis)
4. [Political Science Applications](#political-science-applications)
5. [Journalism and Media](#journalism-and-media)
6. [Academic Research](#academic-research)

---

## Basic Usage

### Getting Started

First, ensure your MCP client is configured to use the Sejm server. Here are examples using JSON-RPC format that any MCP client can understand.

### Example 1: List Current MPs

```json
{
  "tool": "sejm_get_mps",
  "arguments": {
    "term": "10"
  }
}
```

**Response** (shortened for brevity):
```json
[
  {
    "id": 1,
    "firstName": "Marek",
    "lastName": "Kuchciński",
    "club": "Prawo i Sprawiedliwość",
    "active": true,
    "districtName": "Rzeszów",
    "voivodeship": "podkarpackie"
  },
  {
    "id": 2,
    "firstName": "Elżbieta",
    "lastName": "Witek",
    "club": "Prawo i Sprawiedliwość",
    "active": true,
    "districtName": "Kielce",
    "voivodeship": "świętokrzyskie"
  }
]
```

### Example 2: Search Polish Constitution

```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "konstytucja",
    "publisher": "DU",
    "limit": "5"
  }
}
```

**Response**:
```json
{
  "items": [
    {
      "eli": "http://eli.gov.pl/eli/DU/1997/78/pol/t",
      "title": "Konstytucja Rzeczypospolitej Polskiej",
      "publisher": "DU",
      "year": 1997,
      "position": 78,
      "actDate": "1997-04-02",
      "publishDate": "1997-04-16",
      "status": "obowiązujący"
    }
  ],
  "count": 1
}
```

---

## Parliamentary Research

### Analyzing Committee Structure

**Goal**: Understand how parliamentary committees are organized and who leads them.

```json
{
  "tool": "sejm_get_committees",
  "arguments": {
    "term": "10"
  }
}
```

**Use the response to**:
- Map committee hierarchies
- Identify committee chairs and their party affiliations
- Analyze gender representation in leadership roles
- Track committee size and membership distribution

### Voting Pattern Analysis

**Goal**: Analyze voting behavior on specific legislation.

```json
{
  "tool": "sejm_search_votings",
  "arguments": {
    "term": "10",
    "limit": "100"
  }
}
```

**Analysis possibilities**:
- Calculate party discipline scores
- Identify controversial votes (close margins)
- Track abstention patterns
- Analyze coalition stability

### MP Activity Assessment

**Goal**: Evaluate individual MP engagement and effectiveness.

**Step 1**: Get MP details
```json
{
  "tool": "sejm_get_mp_details",
  "arguments": {
    "term": "10",
    "mp_id": "1"
  }
}
```

**Step 2**: Check their interpellations
```json
{
  "tool": "sejm_get_interpellations",
  "arguments": {
    "term": "10",
    "limit": "200"
  }
}
```

**Analysis**: Filter interpellations by MP ID to assess:
- Question volume and topics
- Government response rates
- Follow-up question patterns

---

## Legal Document Analysis

### Constitutional Law Research

**Goal**: Study constitutional amendments and their impact.

**Step 1**: Find constitutional acts
```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "konstytucja",
    "publisher": "DU"
  }
}
```

**Step 2**: Get constitutional text
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

**Step 3**: Explore amendments
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

**Research applications**:
- Timeline of constitutional changes
- Impact analysis of amendments
- Citation network mapping
- Comparative constitutional law studies

### Legislative History Tracking

**Goal**: Track how a specific law has evolved over time.

**Step 1**: Search for related acts
```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "kodeks pracy",
    "publisher": "DU",
    "limit": "50"
  }
}
```

**Step 2**: For each relevant act, get references
```json
{
  "tool": "eli_get_act_references",
  "arguments": {
    "publisher": "DU",
    "year": "1974",
    "position": "24"
  }
}
```

**Analysis outcomes**:
- Legislative genealogy mapping
- Amendment frequency analysis
- Policy stability assessment
- Regulatory burden evaluation

### Legal Citation Analysis

**Goal**: Understand which laws are most frequently referenced.

**Approach**:
1. Systematically query `eli_get_act_references` for major acts
2. Build citation network database
3. Apply network analysis algorithms
4. Identify central/influential legislation

---

## Political Science Applications

### Coalition Government Analysis

**Research Question**: How stable is the current governing coalition?

**Data Collection**:
```json
{
  "tool": "sejm_search_votings",
  "arguments": {
    "term": "10",
    "limit": "500"
  }
}
```

**Analysis Framework**:
- Calculate coalition unity scores
- Identify defection patterns
- Map issue-specific alliances
- Predict coalition stability

### Electoral District Representation

**Research Question**: How do MPs represent their districts?

**Step 1**: Get MPs by district
```json
{
  "tool": "sejm_get_mps",
  "arguments": {
    "term": "10"
  }
}
```

**Step 2**: Analyze interpellation patterns
```json
{
  "tool": "sejm_get_interpellations",
  "arguments": {
    "term": "10",
    "limit": "1000"
  }
}
```

**Research outputs**:
- Geographic representation analysis
- Local vs national issue focus
- Constituency service patterns
- Regional political differences

### Parliamentary Efficiency Study

**Metrics to calculate**:
- Bills passed vs introduced ratio
- Average time from introduction to passage
- Committee processing efficiency
- Amendment success rates

**Data sources**: Combine voting records with committee data

---

## Journalism and Media

### Government Accountability Reporting

**Story Angle**: "Which ministers are slowest to respond to parliamentary questions?"

**Data Collection**:
```json
{
  "tool": "sejm_get_interpellations",
  "arguments": {
    "term": "10",
    "limit": "2000"
  }
}
```

**Analysis**:
- Group by recipient ministry
- Calculate average response times
- Identify chronic delays
- Highlight unanswered questions

**Visualization ideas**:
- Response time heatmap by ministry
- Timeline of delayed responses
- Comparison with previous terms

### Election Preparation Coverage

**Story Angle**: "How active are MPs in their final term year?"

**Metrics**:
- Interpellation volume changes
- Voting attendance patterns
- Committee participation rates
- Local vs national issue focus shifts

### Legislative Impact Stories

**Example**: "New COVID-19 regulations: What changed?"

**Research approach**:
1. Search for pandemic-related legislation
```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "covid",
    "year": "2020"
  }
}
```

2. Get full text for key acts
3. Track amendment history
4. Analyze implementation timeline

---

## Academic Research

### Comparative Politics Research

**Project**: "Parliamentary Questions in European Democracies"

**Polish data contribution**:
- Interpellation volume and topics
- Response time analysis
- Government-opposition dynamics
- Temporal trends analysis

**Cross-national comparison framework**:
- Question types and procedures
- Response quality and timeliness
- Political impact and media coverage

### Legal Informatics Research

**Project**: "Automated Legal Document Classification"

**Dataset creation**:
1. Bulk download legal acts by category
2. Extract structural features
3. Build training/testing datasets
4. Develop classification algorithms

**Applications**:
- Legal document summarization
- Citation recommendation systems
- Regulatory compliance tools
- Legal knowledge graphs

### Policy Network Analysis

**Research Design**: "Policy Community Structures in Polish Parliament"

**Data Collection Strategy**:
1. Map committee memberships over time
2. Track interpellation co-sponsorship
3. Analyze voting coalitions
4. Identify policy entrepreneurs

**Network Analysis Methods**:
- Centrality measures for MP influence
- Community detection algorithms
- Temporal network evolution
- Cross-committee collaboration patterns

---

## Advanced Integration Examples

### Multi-API Workflow

**Goal**: Create comprehensive MP profiles combining parliamentary and legal data.

```python
# Pseudo-code for complex analysis
def analyze_mp_legal_expertise(mp_id, term):
    # Get MP details
    mp_data = sejm_get_mp_details(term=term, mp_id=mp_id)

    # Get their interpellations
    interpellations = sejm_get_interpellations(term=term, limit=1000)
    mp_interpellations = filter_by_author(interpellations, mp_id)

    # Extract legal topics from interpellation titles
    legal_topics = extract_legal_topics(mp_interpellations)

    # For each topic, find relevant legislation
    for topic in legal_topics:
        related_acts = eli_search_acts(title=topic, limit=20)

        # Analyze MP's expertise depth
        expertise_score = calculate_expertise(
            mp_interpellations,
            related_acts
        )

    return compile_expertise_profile(mp_data, expertise_scores)
```

### Real-time Monitoring System

**Architecture for live parliamentary tracking**:

```python
class ParliamentaryMonitor:
    def __init__(self):
        self.sejm_client = SejmMCPClient()
        self.eli_client = ELIMCPClient()

    def daily_update(self):
        # Check for new interpellations
        new_interpellations = self.check_new_interpellations()

        # Monitor voting schedules
        upcoming_votes = self.check_voting_schedule()

        # Track legal publication updates
        new_acts = self.check_new_legislation()

        # Generate alerts and reports
        self.generate_daily_report(
            new_interpellations,
            upcoming_votes,
            new_acts
        )
```

### Academic Citation Tool

**Tool for generating proper legal citations**:

```python
def generate_citation(publisher, year, position, format='apa'):
    # Get act details
    act_details = eli_get_act_details(
        publisher=publisher,
        year=year,
        position=position
    )

    # Format according to academic standards
    if format == 'apa':
        return format_apa_citation(act_details)
    elif format == 'chicago':
        return format_chicago_citation(act_details)
    elif format == 'mla':
        return format_mla_citation(act_details)
```

---

## Performance Optimization Tips

### Batch Processing

When analyzing large datasets:
- Use appropriate `limit` parameters
- Implement local caching for reference data
- Process data in chunks to avoid timeouts
- Cache frequently accessed information

### Error Handling

```python
def robust_api_call(tool_name, arguments, max_retries=3):
    for attempt in range(max_retries):
        try:
            return mcp_client.call_tool(tool_name, arguments)
        except TimeoutError:
            if attempt < max_retries - 1:
                time.sleep(2 ** attempt)  # Exponential backoff
                continue
            raise
        except APIError as e:
            if "rate limit" in str(e).lower():
                time.sleep(60)  # Wait for rate limit reset
                continue
            raise
```

### Data Validation

Always validate API responses:
- Check for required fields
- Validate data types and ranges
- Handle missing or null values gracefully
- Implement data quality checks

---

## Conclusion

The Sejm MCP Server provides rich access to Polish parliamentary and legal data. These examples demonstrate the breadth of research and analysis possibilities, from simple queries to complex multi-API workflows.

For more specific use cases or custom implementations, refer to the [API Reference](API_REFERENCE.md) for detailed parameter and response documentation.