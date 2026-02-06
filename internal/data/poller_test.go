package data

import (
	"testing"
	"time"
)

func TestScheduleStatusPoll(t *testing.T) {
	cmd := ScheduleStatusPoll(5 * time.Second)
	if cmd == nil {
		t.Fatal("ScheduleStatusPoll returned nil cmd")
	}
}

func TestScheduleAgentPoll(t *testing.T) {
	cmd := ScheduleAgentPoll(3 * time.Second)
	if cmd == nil {
		t.Fatal("ScheduleAgentPoll returned nil cmd")
	}
}

func TestScheduleConvoyPoll(t *testing.T) {
	cmd := ScheduleConvoyPoll(10 * time.Second)
	if cmd == nil {
		t.Fatal("ScheduleConvoyPoll returned nil cmd")
	}
}

func TestFetchStatusCmdReturnsMsg(t *testing.T) {
	f := &Fetcher{TownRoot: t.TempDir()}
	cmd := FetchStatusCmd(f)
	if cmd == nil {
		t.Fatal("FetchStatusCmd returned nil cmd")
	}

	msg := cmd()
	update, ok := msg.(StatusUpdateMsg)
	if !ok {
		t.Fatalf("expected StatusUpdateMsg, got %T", msg)
	}
	// gt is not on PATH in test env, so we expect an error
	if update.Err == nil && update.Status == nil {
		t.Error("expected either data or error")
	}
}

func TestFetchAgentsCmdReturnsMsg(t *testing.T) {
	f := &Fetcher{TownRoot: t.TempDir()}
	cmd := FetchAgentsCmd(f)
	if cmd == nil {
		t.Fatal("FetchAgentsCmd returned nil cmd")
	}

	msg := cmd()
	update, ok := msg.(AgentUpdateMsg)
	if !ok {
		t.Fatalf("expected AgentUpdateMsg, got %T", msg)
	}
	if update.Err == nil && update.Sessions == nil {
		t.Error("expected either data or error")
	}
}

func TestFetchConvoysCmdReturnsMsg(t *testing.T) {
	f := &Fetcher{TownRoot: t.TempDir()}
	cmd := FetchConvoysCmd(f)
	if cmd == nil {
		t.Fatal("FetchConvoysCmd returned nil cmd")
	}

	msg := cmd()
	update, ok := msg.(ConvoyUpdateMsg)
	if !ok {
		t.Fatalf("expected ConvoyUpdateMsg, got %T", msg)
	}
	if update.Err == nil && update.Convoys == nil {
		t.Error("expected either data or error")
	}
}

func TestFetchStatusCmdNoPanic(t *testing.T) {
	f := &Fetcher{TownRoot: "/nonexistent/path"}
	cmd := FetchStatusCmd(f)
	// Should not panic regardless of whether gt is available
	msg := cmd()
	if _, ok := msg.(StatusUpdateMsg); !ok {
		t.Fatalf("expected StatusUpdateMsg, got %T", msg)
	}
}

func TestFetchAgentsCmdNoPanic(t *testing.T) {
	f := &Fetcher{TownRoot: "/nonexistent/path"}
	cmd := FetchAgentsCmd(f)
	msg := cmd()
	if _, ok := msg.(AgentUpdateMsg); !ok {
		t.Fatalf("expected AgentUpdateMsg, got %T", msg)
	}
}

func TestFetchConvoysCmdNoPanic(t *testing.T) {
	f := &Fetcher{TownRoot: "/nonexistent/path"}
	cmd := FetchConvoysCmd(f)
	msg := cmd()
	if _, ok := msg.(ConvoyUpdateMsg); !ok {
		t.Fatalf("expected ConvoyUpdateMsg, got %T", msg)
	}
}
