package gizclaw

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/genx"
	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/api/rpcapi"
)

func TestGenXRewardDeciderInvokesConfiguredPattern(t *testing.T) {
	gen := &systemTaskGenerator{args: `{"badge_id":"founder","point_amount":12}`}
	decision, err := (genxRewardDecider{Generator: gen, Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "won a race")
	if err != nil {
		t.Fatalf("DecideReward() error = %v", err)
	}
	if gen.pattern != "model/reward" {
		t.Fatalf("pattern = %q, want model/reward", gen.pattern)
	}
	if gen.tool != "decide_reward" {
		t.Fatalf("tool = %q, want decide_reward", gen.tool)
	}
	if decision.BadgeId != "founder" || decision.PointAmount != 12 {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestGenXPetActionDeciderInvokesConfiguredPattern(t *testing.T) {
	gen := &systemTaskGenerator{args: `{"point_delta":-1,"life_delta":{"satiety":5},"ability_delta":{"exp":3}}`}
	decision, err := (genxPetActionDecider{Generator: gen, Pattern: "model/pet"}).DecidePetAction(context.Background(), "feed", "ate lunch", rpcapi.PetObject{Id: "pet-a"})
	if err != nil {
		t.Fatalf("DecidePetAction() error = %v", err)
	}
	if gen.pattern != "model/pet" {
		t.Fatalf("pattern = %q, want model/pet", gen.pattern)
	}
	if gen.tool != "decide_pet_action" {
		t.Fatalf("tool = %q, want decide_pet_action", gen.tool)
	}
	if decision.PointDelta != -1 || decision.LifeDelta.Satiety != 5 || decision.AbilityDelta.Exp != 3 {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestSystemTaskDecisionRejectsMissingGeneratorResult(t *testing.T) {
	_, err := (genxRewardDecider{Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "claim")
	if err == nil {
		t.Fatal("DecideReward(nil generator) error = nil")
	}

	_, err = (genxRewardDecider{Generator: &systemTaskGenerator{}, Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "claim")
	if err == nil {
		t.Fatal("DecideReward(no call) error = nil")
	}

	want := errors.New("invoke failed")
	_, err = (genxRewardDecider{Generator: &systemTaskGenerator{err: want}, Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "claim")
	if !errors.Is(err, want) {
		t.Fatalf("DecideReward(invoke error) = %v, want %v", err, want)
	}

	_, err = (genxRewardDecider{Generator: &systemTaskGenerator{args: `{"point_amount":1}`, toolName: "other_tool"}, Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "claim")
	if err == nil || !strings.Contains(err.Error(), "unsupported tool call") {
		t.Fatalf("DecideReward(wrong tool) error = %v, want unsupported tool call", err)
	}

	_, err = (genxRewardDecider{Generator: &systemTaskGenerator{args: `{"point_amount":`}, Pattern: "model/reward"}).DecideReward(context.Background(), "gear-a", "claim")
	if err == nil || !strings.Contains(err.Error(), "parse system task decision") {
		t.Fatalf("DecideReward(invalid JSON) error = %v, want parse error", err)
	}
}

type systemTaskGenerator struct {
	pattern  string
	tool     string
	args     string
	toolName string
	err      error
}

func (*systemTaskGenerator) GenerateStream(context.Context, string, genx.ModelContext) (genx.Stream, error) {
	return nil, errors.New("unused")
}

func (g *systemTaskGenerator) Invoke(_ context.Context, pattern string, _ genx.ModelContext, tool *genx.FuncTool) (genx.Usage, *genx.FuncCall, error) {
	g.pattern = pattern
	if tool != nil {
		g.tool = tool.Name
	}
	if g.err != nil {
		return genx.Usage{}, nil, g.err
	}
	if g.args == "" {
		return genx.Usage{}, nil, nil
	}
	if g.toolName != "" {
		return genx.Usage{}, &genx.FuncCall{Name: g.toolName, Arguments: g.args}, nil
	}
	return genx.Usage{}, tool.NewFuncCall(g.args), nil
}
