# Analysis Framework for TraceIQ & Timeline Systems

## Phase 1: Independent Analysis (Both Agents)

Each agent must independently analyze ALL extracted code and documentation to answer:

### Application Intent Analysis
1. **What problem is this solving?**
   - What user need is being addressed?
   - What pain points does this eliminate?
   - What workflow does this enable?

2. **How should the application work?**
   - Core workflow from data ingestion to output
   - User interactions and interfaces
   - Data transformations and processing steps
   - Integration points with external systems

3. **Data Model Requirements**
   - What entities exist in the domain?
   - What relationships connect them?
   - What attributes are essential vs. optional?
   - What queries will be run most frequently?

### Technical Architecture Analysis
4. **Data Sources Identified**
   - Google Timeline/Location History
   - Chat conversations (multiple platforms)
   - Legal/court documents
   - Personal life events
   - Other sources?

5. **Processing Pipeline**
   - Ingestion methods
   - Transformation logic
   - Validation rules
   - Enrichment processes (geocoding, etc.)
   - Storage strategies

6. **Schema Design Patterns**
   - What tables are needed?
   - What indexes for performance?
   - What constraints for data integrity?
   - What triggers/functions for automation?

### User Goals Analysis
7. **What does the user want to achieve?**
   - Timeline reconstruction from multiple sources
   - Evidence gathering and organization
   - Pattern detection across life events
   - Relationship mapping
   - Location verification

8. **What features are critical vs. nice-to-have?**
   - Must-have for MVP
   - Important for usability
   - Future enhancements

### Schema Comparison
9. **Version Analysis**
   - Which schema version is most complete?
   - What evolved between versions?
   - What was added, removed, refined?
   - Which represents the current vision?

10. **Integration Requirements**
    - How do TraceIQ, Timeline, Chat, and Personal History fit together?
    - Shared tables? Separate schemas with links?
    - Unified timeline view?

## Phase 2: Consensus Building

After both agents complete their independent analyses:

1. **Compare Understanding**
   - Where do both agents agree?
   - Where do they differ?
   - What did each agent catch that the other missed?

2. **Synthesize Best Elements**
   - Best schema design choices
   - Most complete feature set
   - Optimal architecture

3. **Create Unified Vision**
   - Single coherent description of what to build
   - Complete schema that serves all use cases
   - Clear implementation plan

## Output Format

Each agent should produce:
```markdown
# [Agent Name] Independent Analysis

## User Intent (What & Why)
[Analysis]

## Application Workflow (How)
[Analysis]

## Data Model Requirements
[Analysis]

## Schema Recommendations
[Specific tables, columns, indexes, constraints]

## Critical vs. Optional Features
[Analysis]

## Version Assessment
[Which schemas are most complete/current]

## Open Questions
[What's unclear or needs user clarification]
```
