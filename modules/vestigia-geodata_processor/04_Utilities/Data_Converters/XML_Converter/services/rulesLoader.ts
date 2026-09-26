/**
 * Rules Loader - Loads YAML rule files from ConflictAnalysisApp
 *
 * Loads behavioral patterns, entities, and other rules for surface-level
 * micro-analysis during ingestion. These are simple pattern matches -
 * macro analysis happens later.
 */

export interface BehaviorRule {
    pattern: string;
    regex: boolean;
}

export interface EntityDefinition {
    name: string;
    aliases?: string[];
}

export interface RulesData {
    behaviors: Record<string, (string | BehaviorRule)[]>;
    entities: {
        people: EntityDefinition[];
        places: EntityDefinition[];
        platforms: string[];
        substances: Record<string, string[]>;
    };
}

/**
 * Load rules from YAML files
 * For now, we'll embed the critical rules directly
 * Later: Load from actual YAML files using a YAML parser
 */
export class RulesLoader {
    private static instance: RulesLoader;
    private rules: RulesData | null = null;
    private rulesPath: string;

    private constructor(rulesPath: string = './ConflictAnalysisApp/rules') {
        this.rulesPath = rulesPath;
    }

    /**
     * Get singleton instance
     * @param rulesPath - Optional path to rules folder (for different cases/projects)
     */
    static getInstance(rulesPath?: string): RulesLoader {
        if (!RulesLoader.instance || (rulesPath && rulesPath !== RulesLoader.instance.rulesPath)) {
            RulesLoader.instance = new RulesLoader(rulesPath);
        }
        return RulesLoader.instance;
    }

    /**
     * Load rules (embedded for now - will add YAML parsing later)
     */
    async loadRules(): Promise<RulesData> {
        if (this.rules) return this.rules;

        // TODO: Load from actual YAML files in ConflictAnalysisApp/rules/
        // For now, embed ALL patterns from behaviors.yaml (21 categories, 140+ patterns)
        this.rules = {
            behaviors: {
                manipulation_gaslighting: [
                    "that's not how it happened",
                    "that never happened",
                    "you're being dramatic",
                    "you're imagining things",
                    "you misremember",
                    "i was just kidding",
                    { pattern: "you always", regex: false },
                    { pattern: "you never", regex: false }
                ],
                projection_blameshift: [
                    "you're the liar",
                    "you have anger issues",
                    "you're trying to manipulate me",
                    "you made me do it",
                    "if you hadn't"
                ],
                victimhood_narrative: [
                    "after all i've done for you",
                    "i'm the one suffering",
                    "everyone sees what you do"
                ],
                certainty_absolutism: [
                    "always",
                    "never",
                    "nothing",
                    "everything",
                    "obviously",
                    "clearly",
                    "literally",
                    "completely"
                ],
                character_assassination: [
                    "good for nothing",
                    "ain't shit",
                    "bitch made",
                    "motherfucker",
                    "piece of shit",
                    "psycho",
                    "unstable",
                    "pathetic",
                    "loser",
                    "freak",
                    "weirdo",
                    { pattern: "sick fagot|sick faggot|sick fagget", regex: true },
                    { pattern: "fagot|faggot|fagget", regex: true }
                ],
                coercive_control: [
                    "do not contact",
                    "don't text me",
                    "blocked",
                    "change my number",
                    "talk to me only when",
                    "restricted"
                ],
                stonewalling_blocking: [
                    "i'm done",
                    "this conversation is over",
                    "stop messaging me",
                    "don't contact me"
                ],
                last_minute_plan_change: [
                    "change of plans",
                    "can't make it",
                    "have to cancel",
                    "something came up",
                    "not going to work"
                ],
                parental_alienation: [
                    "doesn't want to see you",
                    "i won't let you see",
                    "you can't see her",
                    "not your business",
                    "only person i can rely on"
                ],
                financial_abuse: [
                    "you can't manage money",
                    "i'm not paying for that",
                    "what do i get out of this",
                    "it's your responsibility to provide"
                ],
                healthcare_denial: [
                    "you don't need a doctor",
                    "therapy is for weak people",
                    "no meds"
                ],
                sexual_weaponization: [
                    "slut",
                    "whore",
                    "pervert",
                    "disgusting",
                    "nasty",
                    "freak",
                    "cheap",
                    "used up",
                    { pattern: "to think i ever did .* with you", regex: true }
                ],
                love_bombing: [
                    "perfect",
                    "amazing",
                    "soulmate",
                    "can't live without you",
                    "forever",
                    "everything",
                    "only one who understands me"
                ],
                feigned_incompetence: [
                    "i'm not smart like you",
                    "i never graduated",
                    "i don't know how",
                    "i don't understand",
                    "just tell me what to do"
                ],
                social_media_deception: [
                    "snapchat",
                    "i don't even use snapchat",
                    "instagram",
                    "facebook",
                    "tiktok",
                    "i never send pictures",
                    "you can check my phone",
                    "nudes",
                    "selfie"
                ],
                infidelity_betrayal: [
                    "cheating",
                    "cheated",
                    "affair",
                    "slept with",
                    "just a friend",
                    "we just work together",
                    "loyal",
                    "faithful"
                ],
                substance_weaponization: [
                    "crackhead",
                    "tweaker",
                    "addict",
                    "junkie",
                    "you're just high",
                    "are you on something",
                    "no one will believe a tweaker"
                ],
                stimulant_control: [
                    "adderall",
                    "addy",
                    "pills",
                    "script",
                    "how many did you take",
                    "you're taking too many",
                    "i only took one",
                    "i'm holding onto them for you"
                ],
                alcohol_risk_child: [
                    "drink",
                    "drank",
                    "drunk",
                    "buzzed",
                    "tipsy",
                    "wasted",
                    "wine",
                    "beer",
                    "liquor",
                    "vodka",
                    "tequila",
                    "hungover",
                    "fireball",
                    { pattern: "i'm fine to drive|i can handle it|i only had one", regex: true }
                ],
                reactive_abuse_gotcha: [
                    "see, you're losing it",
                    "this is what i have to deal with",
                    "you're the crazy one",
                    "i'm going to record this",
                    "you're having an episode"
                ],
                dismissiveness: [
                    "whatever",
                    "idc",
                    "do what you want",
                    "not my problem",
                    "grow up",
                    "k.",
                    "k"
                ]
            },
            entities: {
                people: [
                    { name: "Katrina", aliases: ["Kat"] },
                    { name: "Salem", aliases: ["Matt", "Matthew"] }
                ],
                places: [
                    { name: "Huckleberry Junction", aliases: ["Huckleberry", "Huck", "Huck's"] }
                ],
                platforms: [
                    "snapchat",
                    "instagram",
                    "facebook",
                    "tiktok"
                ],
                substances: {
                    alcohol: ["fireball", "vodka", "tequila", "wine", "beer"],
                    stimulants: ["adderall", "addy", "amphetamine"]
                }
            }
        };

        return this.rules;
    }

    /**
     * Get all behavior patterns
     */
    async getBehaviorPatterns(): Promise<Record<string, (string | BehaviorRule)[]>> {
        const rules = await this.loadRules();
        return rules.behaviors;
    }

    /**
     * Get entity definitions
     */
    async getEntities(): Promise<RulesData['entities']> {
        const rules = await this.loadRules();
        return rules.entities;
    }

    /**
     * Check if text matches any pattern in a category
     */
    matchesPattern(text: string, patterns: (string | BehaviorRule)[]): { matched: boolean; matchedText?: string } {
        const lowerText = text.toLowerCase();

        for (const pattern of patterns) {
            if (typeof pattern === 'string') {
                if (lowerText.includes(pattern.toLowerCase())) {
                    return { matched: true, matchedText: pattern };
                }
            } else if (pattern.regex) {
                const regex = new RegExp(pattern.pattern, 'i');
                if (regex.test(text)) {
                    return { matched: true, matchedText: pattern.pattern };
                }
            } else {
                if (lowerText.includes(pattern.pattern.toLowerCase())) {
                    return { matched: true, matchedText: pattern.pattern };
                }
            }
        }

        return { matched: false };
    }
}
