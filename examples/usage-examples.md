# Sejm MCP Server Usage Examples

This document provides practical examples of how to use the Sejm MCP Server with AI assistants to query Polish Parliament data.

## Basic Queries

### Getting MPs Information

```
Show me all current Members of Parliament from the ruling party
```

```
Get details about MP with ID 123 from the current term
```

```
List all MPs from Warsaw electoral district
```

### Parliamentary Committees

```
What committees exist in the current parliamentary term?
```

```
Show me the composition of the Finance Committee
```

### Voting Records

```
Show me the most recent 20 voting records from the current term
```

```
Get voting results from sitting number 45
```

```
Analyze voting patterns for controversial legislation
```

### Interpellations (Parliamentary Questions)

```
Show me recent interpellations to the Minister of Health
```

```
Get all interpellations from the last month
```

```
Find interpellations about climate change policies
```

## Legal Research with ELI API

### Searching Legal Acts

```
Find the Polish Constitution in the legal database
```

```
Search for laws related to data protection (GDPR implementation)
```

```
Look up employment law regulations from 2020
```

### Getting Legal Document Details

```
Get full details about the act DU/1997/78 (Polish Constitution)
```

```
Show me the metadata for legal act DU/2016/538
```

### Legal Document Text

```
Get the full text of the Polish Constitution in HTML format
```

```
Download the PDF version of the Labor Code
```

### Legal References and Relationships

```
Show me what laws reference the Polish Constitution
```

```
Find all acts that have been amended by recent legislation
```

```
Map the legal relationship network for environmental protection laws
```

### Publishers and Legal System Navigation

```
What are all the official legal publishers in Poland?
```

```
Show me the structure of the Polish legal publication system
```

## Advanced Analysis Queries

### Political Analysis

```
Compare voting attendance between different political parties
```

```
Analyze which MPs ask the most interpellations
```

```
Show committee membership distribution by party affiliation
```

### Legislative Tracking

```
Track the legislative process for bills related to digital rights
```

```
Monitor government responses to parliamentary questions about healthcare
```

### Legal Research

```
Find all laws that have been repealed in the last 5 years
```

```
Show the evolution of privacy laws in Poland
```

```
Compare current employment law with previous versions
```

## Data Export and Processing

### Getting Structured Data

```
Export all MP data as JSON for analysis
```

```
Get committee structure data for visualization
```

### Statistical Analysis

```
Calculate average response time for government replies to interpellations
```

```
Analyze voting success rates by party
```

```
Generate statistics on legislative activity by term
```

## Research Scenarios

### Academic Research

**Scenario**: Studying Polish parliamentary democracy
```
Get comprehensive data on parliamentary activity for the current term including:
- All MPs with their biographical data
- Committee structures and memberships
- Voting records and patterns
- Government accountability through interpellations
```

### Journalism and Media

**Scenario**: Investigating government responsiveness
```
Analyze government response times to parliamentary questions:
- Get all interpellations from the last 6 months
- Check which have received responses
- Calculate average response delays
- Identify ministries with longest delays
```

### Legal Professional

**Scenario**: Legal research and case preparation
```
Research employment law:
- Find all employment-related legal acts
- Get the current Labor Code text
- Check recent amendments and changes
- Map references between employment laws
```

### Civic Engagement

**Scenario**: Understanding your representation
```
Find information about your local representatives:
- Get MPs from your electoral district
- See their committee memberships
- Check their voting records
- Review their parliamentary questions
```

## Integration Tips

### Combining Multiple Queries

The MCP server allows you to chain queries for comprehensive analysis:

1. **Start broad**: "Get all MPs from the current term"
2. **Narrow down**: "Show me MPs from the Finance Committee"
3. **Get details**: "Get voting records for these MPs on budget-related votes"
4. **Analyze**: "Calculate their voting alignment on fiscal policy"

### Working with Large Datasets

For large datasets, use pagination and filtering:

```
Get MPs in batches of 50 for processing
```

```
Filter voting records by date range to avoid timeouts
```

### Error Handling

The server provides helpful error messages:
- Invalid term numbers (valid range: 1-10)
- Missing required parameters
- API connectivity issues
- Data parsing problems

## Best Practices

1. **Start with general queries** before drilling down to specifics
2. **Use current term (10)** for most up-to-date data
3. **Check API connectivity** if you encounter errors
4. **Be patient** with large data requests (voting records, interpellations)
5. **Combine Sejm and ELI data** for comprehensive political and legal analysis

## Troubleshooting

### Common Issues

**"Invalid parliamentary term"**
- Use term numbers 1-10, where 10 is current (2019-2023)

**"API request failed"**
- Check internet connectivity
- Verify Polish Parliament API is accessible

**"Failed to retrieve data"**
- Some historical data may not be available
- Try different search parameters

**"No results found"**
- Check spelling of search terms
- Try broader search criteria
- Verify the data exists for the specified term