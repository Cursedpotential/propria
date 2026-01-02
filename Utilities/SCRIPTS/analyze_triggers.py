#!/usr/bin/env python3
"""
Analyze skill trigger quality across all installed marketplaces.
Checks for anti-patterns documented in writing-when-clauses.md
"""

import os
import re
from pathlib import Path
from collections import defaultdict

# Anti-pattern detectors
INTROSPECTION_VERBS = ['notice', 'catch', 'sense', 'realize', 'find yourself', 'become aware']
EMOTIONAL_WORDS = ['overwhelmed', 'stuck', 'confused', 'frustrated', 'feel', 'seems', 'hunch', 'intuition']
VAGUE_TRIGGERS = ['when appropriate', 'when needed', 'when helpful', 'when necessary']
QUALIFIER_PATTERNS = ['particularly if', 'especially when', 'provided that', 'unless', 'which often indicates']

def extract_description(skill_file):
    """Extract description from SKILL.md frontmatter"""
    try:
        with open(skill_file, 'r', encoding='utf-8') as f:
            content = f.read()

        # Find frontmatter
        match = re.search(r'^---\s*\n(.*?)\n---', content, re.DOTALL | re.MULTILINE)
        if not match:
            return None

        frontmatter = match.group(1)

        # Extract description (may be multi-line)
        desc_match = re.search(r'^description:\s*(.+?)(?=\n[a-z_-]+:|$)', frontmatter, re.DOTALL | re.MULTILINE)
        if desc_match:
            return desc_match.group(1).strip()
    except Exception as e:
        return None
    return None

def check_triggers(description, skill_name, marketplace):
    """Check description for trigger anti-patterns"""
    if not description:
        return []

    issues = []
    desc_lower = description.lower()

    # Check for introspection verbs
    for verb in INTROSPECTION_VERBS:
        if verb in desc_lower:
            issues.append({
                'severity': 'HIGH',
                'type': 'introspection',
                'pattern': verb,
                'skill': skill_name,
                'marketplace': marketplace,
                'description': description
            })

    # Check for emotional language
    for word in EMOTIONAL_WORDS:
        if word in desc_lower:
            issues.append({
                'severity': 'MEDIUM',
                'type': 'emotional',
                'pattern': word,
                'skill': skill_name,
                'marketplace': marketplace,
                'description': description
            })

    # Check for vague triggers
    for vague in VAGUE_TRIGGERS:
        if vague in desc_lower:
            issues.append({
                'severity': 'MEDIUM',
                'type': 'vague',
                'pattern': vague,
                'skill': skill_name,
                'marketplace': marketplace,
                'description': description
            })

    # Check for over-qualification
    for qualifier in QUALIFIER_PATTERNS:
        if qualifier in desc_lower:
            issues.append({
                'severity': 'LOW',
                'type': 'over-qualified',
                'pattern': qualifier,
                'skill': skill_name,
                'marketplace': marketplace,
                'description': description
            })

    return issues

def main():
    base_dir = Path('C:/Users/matts/.claude/plugins/marketplaces')

    all_issues = []
    stats = defaultdict(int)
    marketplace_stats = defaultdict(lambda: {'total': 0, 'issues': 0})

    # Scan all marketplaces
    for marketplace_dir in sorted(base_dir.iterdir()):
        if not marketplace_dir.is_dir():
            continue

        marketplace_name = marketplace_dir.name

        # Find all SKILL.md files
        skill_files = list(marketplace_dir.rglob('SKILL.md'))

        for skill_file in skill_files:
            skill_name = skill_file.parent.name
            description = extract_description(skill_file)

            marketplace_stats[marketplace_name]['total'] += 1
            stats['total_skills'] += 1

            if description:
                stats['with_description'] += 1
                issues = check_triggers(description, skill_name, marketplace_name)

                if issues:
                    all_issues.extend(issues)
                    marketplace_stats[marketplace_name]['issues'] += 1
                    stats['skills_with_issues'] += 1
            else:
                stats['missing_description'] += 1

    # Generate Report
    print("=" * 80)
    print("SKILL TRIGGER QUALITY ANALYSIS")
    print("=" * 80)
    print()

    print("SUMMARY STATISTICS")
    print("-" * 80)
    print(f"Total skills analyzed: {stats['total_skills']}")
    print(f"Skills with descriptions: {stats['with_description']}")
    print(f"Skills missing descriptions: {stats['missing_description']}")
    print(f"Skills with trigger issues: {stats['skills_with_issues']}")
    print(f"Total issues found: {len(all_issues)}")
    print()

    # Group by severity
    by_severity = defaultdict(list)
    for issue in all_issues:
        by_severity[issue['severity']].append(issue)

    # HIGH Priority Issues
    if by_severity['HIGH']:
        print("=" * 80)
        print(f"HIGH PRIORITY - Introspection-based triggers ({len(by_severity['HIGH'])} issues)")
        print("=" * 80)
        for issue in by_severity['HIGH']:  # Show all
            print(f"\n{issue['marketplace']}:{issue['skill']}")
            print(f"  Pattern: '{issue['pattern']}'")
            print(f"  Description: {issue['description']}")
        print()

    # MEDIUM Priority Issues
    if by_severity['MEDIUM']:
        print("=" * 80)
        print(f"MEDIUM PRIORITY - Vague/Emotional triggers ({len(by_severity['MEDIUM'])} issues)")
        print("=" * 80)
        for issue in by_severity['MEDIUM']:  # Show all
            print(f"\n{issue['marketplace']}:{issue['skill']}")
            print(f"  Pattern: '{issue['pattern']}'")
            print(f"  Type: {issue['type']}")
        print()

    # LOW Priority Issues
    if by_severity['LOW']:
        print("=" * 80)
        print(f"LOW PRIORITY - Over-qualified triggers ({len(by_severity['LOW'])} issues)")
        print("=" * 80)
        for issue in by_severity['LOW']:
            print(f"\n{issue['marketplace']}:{issue['skill']}")
            print(f"  Pattern: '{issue['pattern']}'")
        print()

    # Marketplace breakdown
    print("=" * 80)
    print("BREAKDOWN BY MARKETPLACE")
    print("=" * 80)
    for marketplace in sorted(marketplace_stats.keys()):
        mstats = marketplace_stats[marketplace]
        if mstats['total'] > 0:
            issue_pct = (mstats['issues'] / mstats['total'] * 100) if mstats['total'] > 0 else 0
            status = "PASS" if mstats['issues'] == 0 else "WARN"
            print(f"[{status}] {marketplace:30s} {mstats['total']:3d} skills, {mstats['issues']:3d} with issues ({issue_pct:5.1f}%)")

    print()
    print("=" * 80)

if __name__ == '__main__':
    main()
