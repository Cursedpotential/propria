# Direct Think Commands

Claude Code exposes all 84 underlying models as manual-only slash commands. The category/router skills remain intact; these commands are an exact-model access layer.

Syntax: `/think:<model-command> [micro|focused|full] [task]`

> `incentives` is the only naming collision. `/think:incentives` remains the family skill; use `/think:incentives-model` for the underlying `incentives` model.

## reason

- `/think:chestertons-fence` — Before removing something, understand why it was put there in the first place—reforms without understanding cause unintended consequences
- `/think:hanlons-razor` — Never attribute to malice that which is adequately explained by incompetence
- `/think:hitchens-razor` — What can be asserted without evidence can be dismissed without evidence
- `/think:humes-guillotine` — You cannot derive an 'ought' from an 'is' without bridging logic
- `/think:poppers-falsifiability` — Scientific theories must be testable and capable of being proven false
- `/think:sagan-standard` — Calibrate evidence requirements proportional to claim implausibility when evaluating extraordinary assertions
- `/think:thinking-first-principles` — Break complex problems into fundamental truths by questioning assumptions and rebuilding from irreducible components. Use for innovation, challenging status quo, or when conventional solutions fail.
- `/think:thinking-occams-razor` — Apply parsimony principle to prefer simpler explanations with fewer assumptions. Use for hypothesis selection in debugging, architecture decisions, and choosing between competing approaches.
- `/think:thinking-scientific-method` — Hypothesis → Prediction → Test → Revise with explicit falsification. Use for debugging, feature experimentation, performance investigation, and A/B testing design.
- `/think:thinking-socratic` — Systematic questioning framework to deepen understanding, challenge assumptions, and uncover hidden beliefs. Use for requirements gathering, debugging, coaching, and critical analysis.
- `/think:thinking-steel-manning` — Argue against the strongest version of opposing positions, not the weakest. Use for design reviews, evaluating alternatives, conflict resolution, and decision validation.
- `/think:thinking-thought-experiment` — Test ideas through hypothetical scenarios when empirical testing is impractical. Use for architecture evaluation, edge case analysis, ethics considerations, and strategy development.

## diagnose

- `/think:current-reality-tree` — A logical diagram that maps cause-effect relationships between undesirable effects to identify root causes that drive most organizational problems
- `/think:fault-tree-analysis` — Top-down deductive methodology for analyzing system failures by mapping logical relationships between faults, subsystems, and redundant safety elements using Boolean logic diagrams
- `/think:fishbone-diagram` — Visually map all potential causes of a problem across structured categories to systematically identify root causes through team collaboration
- `/think:thinking-five-whys-plus` — Enhanced root cause analysis with explicit bias guards and stopping criteria. Use for incident post-mortems, bug investigations, and process failures where standard 5 Whys might mislead.
- `/think:thinking-kepner-tregoe` — Systematic rational process for complex problem analysis, decision making, and risk assessment. Use for high-stakes engineering decisions, root cause analysis beyond 5 Whys, and multi-factor evaluations requiring structured criteria.

## systems

- `/think:causal-loop-diagrams` — Map cause-and-effect relationships with reinforcing/balancing feedback loops to visualize system structure and predict behavior patterns
- `/think:graph-thinking` — Apply graph-based thinking to visualize complex relationships and solve problems non-linearly. Use when mapping dependencies, analyzing systems, exploring interconnected concepts, or designing architectures.
- `/think:thinking-archetypes` — Recognize Senge's Systems Archetypes to diagnose recurring organizational and technical problems, identify why fixes keep failing, and design interventions that address root structure.
- `/think:thinking-feedback-loops` — Analyze systems using Donella Meadows' feedback loop framework to identify reinforcing loops, balancing loops, delays, and leverage points. Use for organizational dynamics, product growth design, debugging runaway or oscillating systems, and finding high-impact interventions.
- `/think:thinking-leverage-points` — Identify where small changes can have large effects using Donella Meadows' hierarchy of system intervention points. Use for strategic decisions, system optimization, and choosing where to focus engineering effort.
- `/think:thinking-systems` — Analyze problems as interconnected systems with feedback loops, emergent behavior, and non-linear effects. Use for debugging complex systems, architecture decisions, and understanding unexpected behavior.
- `/think:thinking-theory-of-constraints` — Identify and manage the bottleneck; improvements elsewhere don't matter until the constraint is addressed. Use for performance optimization, process improvement, and resource allocation.

## forecast

- `/think:calibration` — Align predicted probabilities with actual outcomes - train yourself so things you say are 70% likely actually happen 70% of the time
- `/think:monte-carlo-simulation` — Probabilistic modeling technique that runs thousands of scenarios with random variable inputs to quantify uncertainty and assess risk distributions
- `/think:outside-view-inside-view` — Forecast by comparing to similar past cases (outside view) instead of analyzing unique project details (inside view) - cures planning fallacy
- `/think:sensitivity-analysis` — Systematic testing of how changes in input variables (assumptions) affect output results, revealing which factors drive outcomes and where model risk is concentrated.
- `/think:superforecasting` — Make probabilistic predictions when strategic decisions require accurate risk assessment
- `/think:tetlocks-10-commandments` — Improve forecast accuracy by following Philip Tetlock's 10 evidence-based principles from superforecaster research distinguishing top 2% predictors
- `/think:thinking-bayesian` — Update beliefs systematically based on new evidence using probabilistic reasoning. Use when estimating probabilities, learning from data, or making decisions under uncertainty.
- `/think:thinking-fermi-estimation` — Make order-of-magnitude estimates for unknown quantities by decomposing into known or estimable factors. Use for capacity planning, cost estimation, market sizing, and technical feasibility assessment.
- `/think:thinking-in-bets` — Frame decisions as probability bets instead of certainties - separate decision quality from outcome quality to avoid resulting bias
- `/think:thinking-probabilistic` — Express confidence in ranges, update predictions with new information, and track calibration over time. Use for project estimation, risk assessment, and decision making under uncertainty.
- `/think:wisdom-of-crowds-aggregation` — Combine diverse independent estimates to produce more accurate predictions than any single expert, when conditions support collective intelligence

## incentives

- `/think:compound-interest` — Leverage exponential growth by reinvesting gains to earn returns on returns, whether in finance, skills, relationships, or habits—small consistent improvements compound into transformational outcomes
- `/think:goodharts-law` — Anticipate metric gaming and unintended optimization by recognizing that when a measure becomes a target, it ceases to be a good measure
- `/think:incentives-model` — Predict and shape behavior by understanding the hidden rewards and punishments that drive human action
- `/think:mechanism-design` — Reverse game theory that engineers rules and incentive structures to achieve desired outcomes when participants have private information and self-interested motivations
- `/think:nash-equilibrium` — Identify stable strategic outcomes where no player benefits from unilaterally changing their strategy - predict where competitive dynamics settle
- `/think:principal-agent-problem` — Conflicts arising when one party (agent) makes decisions on behalf of another (principal), but their interests diverge and information is asymmetric
- `/think:prisoners-dilemma` — Recognize situations where individual rational choices lead to collectively worse outcomes - core model for cooperation vs. defection dynamics
- `/think:thinking-opportunity-cost` — Evaluate decisions by what you give up, not just what you gain. Use for resource allocation, prioritization, build vs. buy choices, and technical debt evaluation.
- `/think:thinking-second-order` — Think beyond immediate consequences to second and third-order effects. Use for strategic decisions, policy changes, and avoiding unintended consequences.
- `/think:time-value-of-money` — Value money received today higher than the same amount in the future because present money can be invested to earn returns—use Net Present Value to compare cash flows across time and make rational investment decisions
- `/think:tit-for-tat` — Start new business relationships with trust and investment, but immediately match any breach. Return to cooperation as soon as they do. Make the pattern obvious through transparency.

## decide

- `/think:eisenhower-matrix` — Prioritize tasks by urgency and importance—focus on important but not urgent work to prevent firefighting
- `/think:rice-prioritization` — Scoring framework using Reach, Impact, Confidence, and Effort to objectively prioritize product features and projects
- `/think:schelling-fence` — Strategic precommitment to defend an arbitrary line on a slippery slope. Use when you face a continuum from acceptable to unacceptable and need a conspicuous, defensible boundary.
- `/think:thinking-bounded-rationality` — Apply Herbert Simon's Bounded Rationality and satisficing to make good-enough decisions under real-world constraints. Use for design decisions under time pressure, recognizing cognitive limits, and setting appropriate stopping criteria.
- `/think:thinking-cynefin` — Classify problems by complexity domain (clear, complicated, complex, chaotic) and match approach to domain. Use for choosing methodologies, problem framing, and process design.
- `/think:thinking-inversion` — Approach problems backward by identifying paths to failure, then systematically avoiding them. Use for risk identification, planning, and avoiding obvious mistakes.
- `/think:thinking-lindy-effect` — For non-perishable things, future life expectancy is proportional to current age. Use for technology selection, evaluating frameworks/libraries, and predicting tool longevity.
- `/think:thinking-margin-of-safety` — Build in buffers for unknown unknowns and don't optimize to the edge. Use for capacity planning, deadline estimation, architecture design, and risk management.
- `/think:thinking-ooda` — Rapid decision-making loop for dynamic situations. Use for incident response, competitive scenarios, time-sensitive decisions, and situations requiring quick adaptation.
- `/think:thinking-pre-mortem` — Imagine a project has failed and work backward to identify why. Use at project kickoffs, before major decisions, or when optimism may be obscuring risks.
- `/think:thinking-red-team` — Deliberately attack your own plans, systems, and assumptions to find weaknesses before adversaries or reality does. Use for security review, architecture validation, plan stress-testing, and pre-launch preparation.
- `/think:thinking-regret-minimization` — Project to your future self and ask what you would regret not doing. Use for career decisions, strategic pivots, risk-taking choices, and life-changing decisions.
- `/think:thinking-reversibility` — Classify decisions by reversibility and match decision process to decision type. Use for technology choices, architecture decisions, process changes, and hiring decisions.
- `/think:thinking-via-negativa` — Improve by removal rather than addition. Focus on what to stop doing, eliminate the negative, and subtract complexity. Use for system simplification, process improvement, and feature prioritization.

## strategy

- `/think:good-strategy-bad-strategy` — Craft coherent strategy through diagnosis of core challenge, guiding policy, and coordinated actions when facing complex problems requiring more than goals and aspirations
- `/think:pest-analysis` — Generate PEST analysis plans with strategic questions and Perplexity AI prompts. Use when user mentions PEST analysis, strategic planning, market analysis, external factors, environmental scanning, or needs research questions for political economic social technological factors.
- `/think:pest-analysis-critical` — Critically evaluate PEST synthesis tables for biases, gaps, and oversights. Use when user mentions critical review, bias assessment, gap analysis, or after completing PEST synthesis.
- `/think:pest-analysis-execute` — Execute PEST analysis by processing research results from Perplexity AI prompts. Create synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions PEST results, analysis execution, research synthesis, or after generating PEST analysis plans.
- `/think:pest-analysis-strategy` — Transform PEST analysis into actionable strategy: export to CSV, save to Obsidian, create SWOT, define strategic actions, prioritize, and build work plan. Use after completing PEST critical assessment.
- `/think:pest-analysis-synthesis` — Create PEST synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions synthesis table, strategic recommendations, opportunity threat analysis, or after completing PEST research.
- `/think:playing-to-win` — Five-question strategy cascade framework for making integrated strategic choices from winning aspiration to management systems, from P&G's transformation
- `/think:porters-five-forces` — Systematically assess the competitive intensity and attractiveness of an industry by analyzing five fundamental forces that shape competition and profitability
- `/think:thinking-effectuation` — Start with means, not goals; co-create with partners; leverage contingencies. Use for startup strategy, innovation projects, and uncertain/novel domains where planning is unreliable.
- `/think:thinking-jobs-to-be-done` — Understand what 'job' users hire your product to do, focusing on progress users seek rather than features. Use for product development, feature prioritization, user research, and market positioning.
- `/think:thinking-triz` — Apply TRIZ (Theory of Inventive Problem Solving) methodology to resolve technical contradictions and find innovative solutions. Use for engineering design, breaking through impossible constraints, and systematic innovation.
- `/think:three-horizons-framework` — Balance current business performance with emerging opportunities and future innovation by allocating resources across three time horizons of growth

## debias

- `/think:anchoring` — First information disproportionately influences judgments and decisions, even when the initial value is arbitrary or irrelevant
- `/think:availability-heuristic` — Correct probability and frequency judgments distorted by mentally available examples by seeking base rate data before relying on vivid anecdotes
- `/think:confirmation-bias` — Combat the tendency to seek evidence supporting existing beliefs by actively searching for disconfirming data and treating opinions as testable hypotheses
- `/think:dunning-kruger-effect` — Low-skill individuals overestimate their competence because they lack the metacognitive ability to recognize their own incompetence
- `/think:fundamental-attribution-error` — Over-attributing others' behavior to their character while under-weighting situational factors that shape their actions
- `/think:halo-effect` — One positive trait creates a global positive impression, influencing judgments about unrelated characteristics
- `/think:hindsight-bias` — The tendency to perceive past events as more predictable than they actually were, leading to distorted learning and overconfidence
- `/think:sunk-cost-fallacy` — Continuing an endeavor due to previously invested resources (time, money, effort) rather than current and future costs and benefits, escalating commitment to failing courses of action
- `/think:thinking-circle-of-competence` — Know the boundaries of your expertise and operate within them. Use when evaluating opportunities, making decisions outside your domain, or assessing when to defer to experts.
- `/think:thinking-debiasing` — Systematic checklist to identify and counteract cognitive biases in decision-making. Use before major decisions, when evaluating recommendations, or when stakes are high.
- `/think:thinking-dual-process` — Apply Kahneman's Dual-Process Theory to recognize when to trust intuition vs engage deliberate analysis. Use for high-stakes decisions, error-prone contexts, or when balancing speed vs accuracy.
- `/think:thinking-map-territory` — Recognize limits of mental models and diagrams. Use when models diverge from reality, debugging expectation mismatches, or questioning abstraction accuracy.
