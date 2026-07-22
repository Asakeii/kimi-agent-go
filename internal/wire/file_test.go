package wire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWireFileAppendsHeaderAndRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session", "wire.jsonl")
	wireFile, err := OpenWireFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if wireFile.Version() != ProtocolVersion {
		t.Fatalf("got version %q, want %q", wireFile.Version(), ProtocolVersion)
	}

	timestamp := time.Unix(1_700_000_000, 125_000_000)
	if err := wireFile.AppendMessageAt(NewTextPart("你好"), timestamp); err != nil {
		t.Fatal(err)
	}
	if err := wireFile.AppendMessageAt(&TurnEnd{}, timestamp.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	if lines[0] != `{"type":"metadata","protocol_version":"1.10"}` {
		t.Fatalf("unexpected metadata: %s", lines[0])
	}
	if strings.Contains(lines[1], `\u`) {
		t.Fatalf("unicode content was escaped: %s", lines[1])
	}

	records, err := wireFile.ReadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Timestamp != 1_700_000_000.125 {
		t.Fatalf("unexpected timestamp: %v", records[0].Timestamp)
	}
	assertTextMessage(t, decodeRecord(t, records[0]), "你好")
	assertTurnEnd(t, decodeRecord(t, records[1]))
}

func TestWireFileUsesExistingProtocolVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wire.jsonl")
	if err := os.WriteFile(path, []byte("\n{\"type\":\"metadata\",\"protocol_version\":\"1.7\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wireFile, err := OpenWireFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if wireFile.Version() != "1.7" {
		t.Fatalf("got version %q, want 1.7", wireFile.Version())
	}
}

func TestWireFileTreatsHeaderlessFileAsLegacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wire.jsonl")
	content := `{"timestamp":1,"message":{"type":"TurnEnd","payload":{}}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	wireFile, err := OpenWireFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if wireFile.Version() != LegacyProtocolVersion {
		t.Fatalf("got version %q, want %q", wireFile.Version(), LegacyProtocolVersion)
	}
	empty, err := wireFile.IsEmpty()
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("headerless record file should not be empty")
	}
}

func TestWireFileSkipsMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wire.jsonl")
	content := strings.Join([]string{
		`{"type":"metadata","protocol_version":"1.10"}`,
		`not-json`,
		`{}`,
		`{"timestamp":1,"message":{"type":"Unknown","payload":{}}}`,
		`{"timestamp":1,"message":{"type":"TurnEnd","payload":{}}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	wireFile, err := OpenWireFile(path)
	if err != nil {
		t.Fatal(err)
	}
	records, err := wireFile.ReadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
}

func decodeRecord(t *testing.T, record MessageRecord) Message {
	t.Helper()
	message, err := record.DecodeMessage()
	if err != nil {
		t.Fatal(err)
	}
	return message
}
