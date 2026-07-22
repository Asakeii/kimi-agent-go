package wire

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ProtocolVersion       = "1.10"
	LegacyProtocolVersion = "1.1"
)

// FileMetadata is the first non-empty line written to a new wire.jsonl file.
type FileMetadata struct {
	Type            string `json:"type"`
	ProtocolVersion string `json:"protocol_version"`
}

// MessageRecord is the persisted representation of one complete Wire message.
type MessageRecord struct {
	Timestamp float64  `json:"timestamp"`
	Message   Envelope `json:"message"`
}

func NewMessageRecord(message Message, timestamp time.Time) (MessageRecord, error) {
	envelope, err := Encode(message)
	if err != nil {
		return MessageRecord{}, err
	}
	return MessageRecord{
		Timestamp: float64(timestamp.UnixNano()) / float64(time.Second),
		Message:   envelope,
	}, nil
}

func (r MessageRecord) DecodeMessage() (Message, error) {
	return Decode(r.Message)
}

// WireFile appends complete messages to, and reads them from, a wire.jsonl file.
type WireFile struct {
	path            string
	protocolVersion string
	mu              sync.Mutex
}

func OpenWireFile(path string) (*WireFile, error) {
	version, err := loadProtocolVersion(path)
	if err != nil {
		return nil, err
	}
	return &WireFile{
		path:            path,
		protocolVersion: version,
	}, nil
}

func (f *WireFile) Path() string {
	return f.path
}

func (f *WireFile) Version() string {
	return f.protocolVersion
}

func (f *WireFile) IsEmpty() (bool, error) {
	file, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("wire: open %s: %w", f.path, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			metadata, _ := decodeMetadata(line)
			if metadata == nil {
				return false, nil
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return true, nil
			}
			return false, fmt.Errorf("wire: read %s: %w", f.path, readErr)
		}
	}
}

func (f *WireFile) AppendMessage(message Message) error {
	return f.AppendMessageAt(message, time.Now())
}

func (f *WireFile) AppendMessageAt(message Message, timestamp time.Time) error {
	record, err := NewMessageRecord(message, timestamp)
	if err != nil {
		return err
	}
	return f.AppendRecord(record)
}

func (f *WireFile) AppendRecord(record MessageRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("wire: create directory for %s: %w", f.path, err)
	}

	info, err := os.Stat(f.path)
	needsHeader := errors.Is(err, os.ErrNotExist) || err == nil && info.Size() == 0
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("wire: stat %s: %w", f.path, err)
	}

	file, err := os.OpenFile(f.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("wire: open %s for append: %w", f.path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if needsHeader {
		metadata := FileMetadata{Type: "metadata", ProtocolVersion: f.protocolVersion}
		if err := encoder.Encode(metadata); err != nil {
			return fmt.Errorf("wire: write metadata: %w", err)
		}
	}
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("wire: write message record: %w", err)
	}
	return nil
}

// ReadRecords returns every valid message record and skips metadata, blank, and malformed lines.
func (f *WireFile) ReadRecords() ([]MessageRecord, error) {
	file, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("wire: open %s: %w", f.path, err)
	}
	defer file.Close()

	var records []MessageRecord
	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			metadata, _ := decodeMetadata(line)
			if metadata == nil {
				if record, err := decodeMessageRecord(line); err == nil {
					records = append(records, record)
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return records, nil
			}
			return nil, fmt.Errorf("wire: read %s: %w", f.path, readErr)
		}
	}
}

func loadProtocolVersion(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return ProtocolVersion, nil
	}
	if err != nil {
		return "", fmt.Errorf("wire: open %s: %w", path, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			metadata, decodeErr := decodeMetadata(line)
			if decodeErr != nil || metadata == nil {
				return LegacyProtocolVersion, nil
			}
			return metadata.ProtocolVersion, nil
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return LegacyProtocolVersion, nil
			}
			return "", fmt.Errorf("wire: read %s: %w", path, readErr)
		}
	}
}

func decodeMetadata(line []byte) (*FileMetadata, error) {
	var metadata FileMetadata
	if err := json.Unmarshal(line, &metadata); err != nil {
		return nil, err
	}
	if metadata.Type != "metadata" || metadata.ProtocolVersion == "" {
		return nil, nil
	}
	return &metadata, nil
}

func decodeMessageRecord(line []byte) (MessageRecord, error) {
	var raw struct {
		Timestamp *float64  `json:"timestamp"`
		Message   *Envelope `json:"message"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return MessageRecord{}, err
	}
	if raw.Timestamp == nil || raw.Message == nil {
		return MessageRecord{}, fmt.Errorf("wire: incomplete message record")
	}
	if _, err := Decode(*raw.Message); err != nil {
		return MessageRecord{}, err
	}
	return MessageRecord{Timestamp: *raw.Timestamp, Message: *raw.Message}, nil
}
