from __future__ import annotations
import re
from typing import Dict, Any, List, Pattern, Tuple
from vaderSentiment.vaderSentiment import SentimentIntensityAnalyzer

class RuleEngine:
    def __init__(self, rules: Dict[str, Any]):
        self.behaviors_raw = rules.get("behaviors", {})
        self.mcl_raw = rules.get("mcl", {})
        self.entities = rules.get("entities", {})
        self.variants = rules.get("spelling_variants", {})
        self.severity = rules.get("severity", {})
        self.parties = rules.get("parties", {})
        # Build alias map for roles
        self.role_aliases = {}
        for role, obj in (self.parties.get("roles") or {}).items():
            for nm in (obj.get("names") or []):
                self.role_aliases[str(nm).strip().lower()] = role

        self._compile()
        try:
            self._sent = SentimentIntensityAnalyzer()
        except Exception:
            self._sent = None

    def _compile_list(self, items: List[Any]) -> List[Tuple[Pattern, Dict[str,Any]]]:
        compiled = []
        for it in items:
            if isinstance(it, dict):
                pat = it.get("pattern", "")
                is_regex = bool(it.get("regex"))
                flags = re.IGNORECASE
                rx = re.compile(pat if is_regex else re.escape(str(pat)), flags)
                meta = {k:v for k,v in it.items() if k != "pattern"}
                compiled.append((rx, meta))
            else:
                rx = re.compile(re.escape(str(it)), re.IGNORECASE)
                compiled.append((rx, {}))
        return compiled

    def _compile(self):
        # Expand [child] tokens before compiling
        tmp_beh = {cat: self._substitute_child_tokens(items) for cat, items in self.behaviors_raw.items()}
        tmp_mcl = {fac: self._substitute_child_tokens(items) for fac, items in self.mcl_raw.items()}
        self.behaviors = {cat: self._compile_list(items) for cat, items in tmp_beh.items()}
        self.mcl = {fac: self._compile_list(items) for fac, items in tmp_mcl.items()}

    def reload(self, rules: Dict[str, Any]):
        self.__init__(rules)

    def tag_behaviors(self, text: str) -> List[str]:
        t = text or ""
        hits = []
        for cat, rules in self.behaviors.items():
            if any(rx.search(t) for rx,_ in rules):
                hits.append(cat)
        return sorted(set(hits))

    def tag_mcl(self, text: str) -> List[str]:
        t = text or ""
        hits = []
        for fac, rules in self.mcl.items():
            if any(rx.search(t) for rx,_ in rules):
                hits.append(fac)
        return sorted(set(hits))

    def sentiment(self, text: str) -> float:
        if not self._sent:
            return 0.0
        return self._sent.polarity_scores(text or "").get("compound", 0.0)
