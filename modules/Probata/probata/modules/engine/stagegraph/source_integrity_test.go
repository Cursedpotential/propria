// Byline: Codex · GPT-6.1 · 2026-10-06. Independent byte check stays outside universal processing.
package stagegraph

import "testing"

// TestSourceIntegrityIsOptionalAndOnlyRequiresRetention proves this operation cannot alter the base stage sequence.
// Inputs: static descriptors. Outputs: dependency/count assertions. Effects: memory only;
// choose to guard independent assessment registration without scheduling any processing.
func TestSourceIntegrityIsOptionalAndOnlyRequiresRetention(t *testing.T) {
	if len(Stages) != 26 {
		t.Fatal("base stage count changed")
	}
	for _, stage := range Stages {
		if stage.ID == AssessSourceIntegrity {
			t.Fatal("integrity added to universal processing")
		}
		for _, dependency := range stage.DependsOn {
			if dependency == AssessSourceIntegrity {
				t.Fatal("base stage depends on integrity")
			}
		}
	}
	found := 0
	for _, stage := range OptionalStages {
		if stage.ID != AssessSourceIntegrity {
			continue
		}
		found++
		if stage.Responsibility != RespVerify || len(stage.DependsOn) != 1 || stage.DependsOn[0] != RetainOriginal {
			t.Fatal("wrong independent assessment dependency")
		}
	}
	if found != 1 {
		t.Fatal("integrity must have one optional descriptor")
	}
}
