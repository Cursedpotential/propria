#!/usr/bin/env python3
"""Fix trigger anti-patterns in skill descriptions"""

import re
from pathlib import Path

fixes = {
    # Emotional/vague patterns to fix
    'seems unnecessary': 'appears unnecessary',
    'seems': 'appears',
    'feel confident': 'have confidence',
    'feel uncertain': 'lack certainty',
    'feel': 'have',
    'stuck on': 'unable to resolve',
    'stuck': 'blocked',
    'hunch': 'pattern',
    'intuition': 'emerging pattern',

    # Over-qualified patterns
    ', particularly if': ' when',
    ', especially when': ' when',
}

def fix_description(desc):
    """Apply fixes to description"""
    original = desc
    for pattern, replacement in fixes.items():
        desc = desc.replace(pattern, replacement)
    return desc, desc != original

def process_skill_file(filepath):
    """Read, fix if needed, and write back"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        # Extract frontmatter
        match = re.search(r'^---\s*\n(.*?)\n---', content, re.DOTALL | re.MULTILINE)
        if not match:
            return False, "No frontmatter"

        frontmatter = match.group(1)
        before_fm = content[:match.start()]
        after_fm = content[match.end():]

        # Extract and fix description
        desc_match = re.search(r'^(description:\s*)(.+?)(?=\n[a-z_-]+:|$)', frontmatter, re.DOTALL | re.MULTILINE)
        if not desc_match:
            return False, "No description"

        desc_prefix = desc_match.group(1)
        desc_content = desc_match.group(2).strip()

        fixed_desc, was_changed = fix_description(desc_content)

        if not was_changed:
            return False, "No changes needed"

        # Replace description in frontmatter
        new_frontmatter = frontmatter[:desc_match.start()] + desc_prefix + fixed_desc + frontmatter[desc_match.end():]
        new_content = before_fm + '---\n' + new_frontmatter + '\n---' + after_fm

        # Write back
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)

        return True, f"Fixed: {desc_content[:60]}... -> {fixed_desc[:60]}..."

    except Exception as e:
        return False, f"Error: {str(e)}"

def main():
    base_dir = Path('C:/Users/matts/.claude/plugins/marketplaces/thinkies')

    print("Fixing trigger anti-patterns...")
    print("=" * 80)

    skill_files = list(base_dir.rglob('SKILL.md'))
    fixed_count = 0

    for skill_file in skill_files:
        changed, msg = process_skill_file(skill_file)
        if changed:
            fixed_count += 1
            skill_name = skill_file.parent.name
            print(f"\n[FIXED] {skill_name}")
            print(f"  {msg}")

    print(f"\n" + "=" * 80)
    print(f"Fixed {fixed_count} skills")

if __name__ == '__main__':
    main()
