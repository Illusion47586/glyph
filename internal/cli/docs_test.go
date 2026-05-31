package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDocsListJSONIncludesAgentGuide(t *testing.T) {
	out, err := executeCLI("docs", "list", "--json")
	if err != nil {
		t.Fatalf("docs list: %v", err)
	}
	var got response
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Type != "docs_list" {
		t.Fatalf("type = %q", got.Type)
	}
	if !strings.Contains(out, `"id": "cli/agent-guide"`) {
		t.Fatalf("docs list missing agent guide:\n%s", out)
	}
	if !strings.Contains(out, `"kind": "cli"`) {
		t.Fatalf("docs list missing kind:\n%s", out)
	}
}

func TestVersionJSONIncludesBuildMetadata(t *testing.T) {
	out, err := executeCLI("version", "--json")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	for _, want := range []string{`"type": "version"`, `"version": "dev"`, `"commit": "unknown"`, `"date": "unknown"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("version output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallCommandIsAvailable(t *testing.T) {
	out, err := executeCLI("install", "--help")
	if err != nil {
		t.Fatalf("install help: %v", err)
	}
	if !strings.Contains(out, "Install glyph into the user PATH") {
		t.Fatalf("install help missing summary:\n%s", out)
	}
}

func TestDocsShowJSONIncludesContent(t *testing.T) {
	out, err := executeCLI("docs", "show", "cli/agent-guide", "--json")
	if err != nil {
		t.Fatalf("docs show: %v", err)
	}
	for _, want := range []string{`"type": "doc"`, `"id": "cli/agent-guide"`, "Agents can use Glyph through the CLI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs show missing %q:\n%s", want, out)
		}
	}
}

func TestSkillsCommandsExposeGlyphSkill(t *testing.T) {
	list, err := executeCLI("skills", "list", "--json")
	if err != nil {
		t.Fatalf("skills list: %v", err)
	}
	if !strings.Contains(list, `"name": "glyph"`) {
		t.Fatalf("skills list missing glyph:\n%s", list)
	}

	show, err := executeCLI("skills", "show", "glyph", "--json")
	if err != nil {
		t.Fatalf("skills show: %v", err)
	}
	for _, want := range []string{`"type": "skill"`, `"name": "glyph"`, "Use Glyph as the source-control system"} {
		if !strings.Contains(show, want) {
			t.Fatalf("skills show missing %q:\n%s", want, show)
		}
	}
}

func TestDocsShowHumanOutputIsMarkdown(t *testing.T) {
	out, err := executeCLI("docs", "show", "overview")
	if err != nil {
		t.Fatalf("docs show human: %v", err)
	}
	if !strings.Contains(out, "Glyph is an agent-native source control system") {
		t.Fatalf("human docs output missing content:\n%s", out)
	}
}

func executeCLI(args ...string) (string, error) {
	cmd := NewRootCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
