from zep_cloud.external_clients.ontology import (
    EntityModel, 
    EntityText, 
    EdgeModel, 
    EntityBoolean, 
    EntityNumber,
    EntityEdgeSourceTarget
)
from pydantic import Field

# ==========================================
# SALEM v. KINZEL - ZEP ONTOLOGY v3
# Cleaned by Claude 2025-12-11
# ==========================================

# ==========================================
# 1. ENTITY TYPES (5 of 10 max)
# ==========================================

class Person(EntityModel):
    """
    Any individual mentioned in the case. 
    Extraction first (who are they?), then analysis (are they safe?).
    NOTE: 'name' is auto-populated by Zep - do not include.
    """
    # Extraction Fields
    relationship_type: EntityText = Field(
        description="friend, family, coworker, romantic_partner, romantic_interest, ex, neighbor, professional", 
        default=None
    )
    connection_to: EntityText = Field(
        description="Who is this person connected to? petitioner, respondent, child, mutual, unknown", 
        default=None
    )
    gender: EntityText = Field(
        description="male, female, unknown", 
        default=None
    )
    
    # Analysis Fields
    risk_level: EntityText = Field(
        description="Safety assessment: safe, unknown, high_risk (felon/armed), transient (short-term partner)", 
        default="unknown"
    )
    is_replacement_candidate: EntityBoolean = Field(
        description="Is this person being used to 'replace' the father?",
        default=None
    )
    role_in_case: EntityText = Field(
        description="Legal role: petitioner, respondent, witness, flying_monkey, neutral", 
        default=None
    )


class Location(EntityModel):
    """
    Physical places. Critical for proving inconsistencies.
    """
    category: EntityText = Field(
        description="residence, party_store, bar, school, medical_facility, court, public_place, friend_house", 
        default=None
    )
    safety_status: EntityText = Field(
        description="safe, unsafe, chaotic, unknown", 
        default=None
    )
    owner: EntityText = Field(
        description="Whose location? (e.g. 'Matt', 'Catrina', 'Dennis')", 
        default=None
    )


class Incident(EntityModel):
    """
    A specific event in time. The 'Iceberg' events.
    """
    event_type: EntityText = Field(
        description="medical_neglect, withholding, drunk_driving, threat, argument, police_contact, exchange, substance_use, financial_coercion", 
        default=None
    )
    date_approx: EntityText = Field(
        description="Approximate date (YYYY-MM-DD)", 
        default=None
    )
    
    # Substance Tracking
    substances_involved: EntityText = Field(
        description="alcohol, amphetamines, cocaine, mdma, cannabis, poly_substance",
        default=None
    )
    context_of_use: EntityText = Field(
        description="maintenance (functional/work), party (chaotic), lethal (driving/endangerment)",
        default=None
    )
    
    # Strategy Tracking
    is_manufactured: EntityBoolean = Field(
        description="Was this crisis staged/exaggerated by Respondent?",
        default=None
    )
    strategic_goal: EntityText = Field(
        description="distract_from_drug_use, sabotage_work, prevent_exchange, garner_sympathy",
        default=None
    )
    trigger_person: EntityText = Field(
        description="Was this triggered by a new partner arrival?",
        default=None
    )
    mcl_factor: EntityText = Field(
        description="Michigan Best Interest Factor: c, f, g, j, k", 
        default=None
    )


class Statement(EntityModel):
    """
    The core atom of evidence. What was said, to whom, about what.
    Replaces 'Fabricated_Narrative' and 'DARVO' entities with queryable fields.
    NOTE: At 10 fields (max). Cannot add more.
    """
    content_summary: EntityText = Field(
        description="Summary of the statement", 
        default=None
    )
    medium: EntityText = Field(
        description="text, email, verbal, court_filing, police_report", 
        default=None
    )
    
    # Fabricated Narrative Detection
    topic_cluster: EntityText = Field(
        description="What story? matt_abuse_narrative, sobriety_claim, victim_stance, good_mother",
        default=None
    )
    told_to: EntityText = Field(
        description="Recipient: matt, her_mother, friend_name, court, facebook_public",
        default=None
    )
    inconsistency_flag: EntityBoolean = Field(
        description="Does this story change when told to different people?",
        default=None
    )
    
    # DARVO Detection
    darvo_stage: EntityText = Field(
        description="deny, attack, reverse_victim, reverse_offender, none",
        default=None
    )
    
    # Analysis
    deception_strategy: EntityText = Field(
        description="misdirection, minimization, projection, fabrication, denial",
        default=None
    )
    contradicts_evidence: EntityBoolean = Field(
        description="Contradicts known hard data (location/photos)?",
        default=None
    )


class Vulnerability(EntityModel):
    """
    Historical traumas Respondent targets (Factor K/F).
    """
    category: EntityText = Field(
        description="past_trauma, insecurity, medical_history, family_loss", 
        default=None
    )
    description: EntityText = Field(
        description="e.g., '2009 Suicide Attempt', 'Mother's Suicide'", 
        default=None
    )
    owner: EntityText = Field(
        description="Petitioner or Respondent", 
        default="Petitioner"
    )


# ==========================================
# 2. EDGE TYPES (8 of 10 max)
# ==========================================

class CoerciveTactic(EdgeModel):
    """Person -> Incident/Statement. Control method."""
    tactic_type: EntityText = Field(
        description="intimidation, isolation, economic_sabotage, suicide_baiting, triangulation", 
        default=None
    )
    financial_mode: EntityText = Field(
        description="conditional_aid, sabotage, dependency_creation",
        default=None
    )


class SpreadsRumor(EdgeModel):
    """Person -> Person (Audience). Smear Campaigns."""
    intent: EntityText = Field(
        description="isolate, ruin_reputation, gain_sympathy, cover_tracks", 
        default=None
    )
    effect_on_listener: EntityText = Field(
        description="believed, skeptical, confused, turned_against_petitioner", 
        default=None
    )


class Contradicts(EdgeModel):
    """
    Gaslighting Detector. 
    Statement -> Statement (Internal)
    Statement -> Incident (Fact)
    """
    discrepancy_type: EntityText = Field(
        description="lie_detected, projection, hypocritical_standard", 
        default=None
    )


class ExposedTo(EdgeModel):
    """Child -> Person/Incident. Factor F (Moral Fitness)."""
    duration: EntityText = Field(
        description="e.g., '3 weeks', 'overnight'", 
        default=None
    )
    impact: EntityText = Field(
        description="regression, distress, sleep_issue, illness", 
        default=None
    )


class Facilitated(EdgeModel):
    """Person -> Person. Factor J (Willingness to facilitate)."""
    action: EntityText = Field(
        description="blocked_access, encouraged_contact, disparaged_parent, refused_meds", 
        default=None
    )
    gatekeeping_subtype: EntityText = Field(
        description="punitive, exclusionary (new partner), possessive",
        default=None
    )


class Exploits(EdgeModel):
    """Statement/Incident -> Vulnerability. Weaponization of trauma."""
    mechanism: EntityText = Field(
        description="triggering_ptsd, shaming, threat_of_recurrence", 
        default=None
    )
    is_microaggression: EntityBoolean = Field(
        default=None
    )


class WasAt(EdgeModel):
    """Person -> Location. Presence tracking."""
    date_approx: EntityText = Field(
        description="When YYYY-MM-DD", 
        default=None
    )
    with_whom: EntityText = Field(
        description="Who else was there", 
        default=None
    )
    source: EntityText = Field(
        description="How we know: text, photo, timeline, admission", 
        default=None
    )


class MadeStatement(EdgeModel):
    """Person -> Statement. Attribution."""
    date_approx: EntityText = Field(
        description="When said YYYY-MM-DD", 
        default=None
    )


# ==========================================
# 3. ONTOLOGY REGISTRATION
# ==========================================

def register_ontology(client):
    """Call this with your Zep client to register the schema."""
    client.graph.set_ontology(
        entities={
            "Person": Person,
            "Location": Location,
            "Incident": Incident,
            "Statement": Statement,
            "Vulnerability": Vulnerability,
        },
        edges={
            "USED_TACTIC": (CoerciveTactic, [
                EntityEdgeSourceTarget(source="Person", target="Incident"), 
                EntityEdgeSourceTarget(source="Person", target="Statement")
            ]),
            "SPREADS_RUMOR": (SpreadsRumor, [
                EntityEdgeSourceTarget(source="Person", target="Person")
            ]),
            "CONTRADICTS": (Contradicts, [
                EntityEdgeSourceTarget(source="Statement", target="Statement"), 
                EntityEdgeSourceTarget(source="Statement", target="Incident")
            ]),
            "EXPOSED_CHILD": (ExposedTo, [
                EntityEdgeSourceTarget(source="Person", target="Person"), 
                EntityEdgeSourceTarget(source="Incident", target="Person")
            ]),
            "AFFECTED_ACCESS": (Facilitated, [
                EntityEdgeSourceTarget(source="Person", target="Person")
            ]),
            "TARGETED_WOUND": (Exploits, [
                EntityEdgeSourceTarget(source="Statement", target="Vulnerability"), 
                EntityEdgeSourceTarget(source="Incident", target="Vulnerability")
            ]),
            "WAS_AT": (WasAt, [
                EntityEdgeSourceTarget(source="Person", target="Location")
            ]),
            "MADE_STATEMENT": (MadeStatement, [
                EntityEdgeSourceTarget(source="Person", target="Statement")
            ]),
        }
    )
    print("Salem ontology registered successfully.")


# ==========================================
# USAGE
# ==========================================
# from zep_cloud.client import Zep
# client = Zep(api_key="YOUR_API_KEY")
# register_ontology(client)
