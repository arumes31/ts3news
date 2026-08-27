package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncodeAbyssEscrowSnapshot writes a normalized envelope. It deliberately does
// not require relational health so an operator can preserve already-corrupt
// state for diagnosis; Verify/Drill reject that state before restoration.
func EncodeAbyssEscrowSnapshot(writer io.Writer, snapshot AbyssEscrowSnapshot) error {
	if writer == nil {
		return errors.New("encoding Abyss escrow snapshot: nil writer")
	}
	if err := validateAbyssEscrowSnapshotEnvelope(&snapshot); err != nil {
		return fmt.Errorf("encoding Abyss escrow snapshot: %w", err)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return fmt.Errorf("encoding Abyss escrow snapshot: %w", err)
	}
	return nil
}

// DecodeAbyssEscrowSnapshot enforces a caller-supplied byte bound and performs
// full checksum and relational validation before returning a snapshot.
func DecodeAbyssEscrowSnapshot(reader io.Reader, maxBytes int64) (AbyssEscrowSnapshot, error) {
	if reader == nil {
		return AbyssEscrowSnapshot{}, errors.New("decoding Abyss escrow snapshot: nil reader")
	}
	if maxBytes <= 0 {
		return AbyssEscrowSnapshot{}, errors.New("decoding Abyss escrow snapshot: invalid byte limit")
	}
	limited := &io.LimitedReader{R: reader, N: maxBytes + 1}
	encoded, err := io.ReadAll(limited)
	if err != nil {
		return AbyssEscrowSnapshot{}, fmt.Errorf("reading Abyss escrow snapshot: %w", err)
	}
	if int64(len(encoded)) > maxBytes {
		return AbyssEscrowSnapshot{}, fmt.Errorf("abyss escrow snapshot exceeds %d-byte limit", maxBytes)
	}
	var snapshot AbyssEscrowSnapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return AbyssEscrowSnapshot{}, fmt.Errorf("decoding Abyss escrow snapshot: %w", err)
	}
	if err := ValidateAbyssEscrowSnapshot(snapshot); err != nil {
		return AbyssEscrowSnapshot{}, fmt.Errorf("verifying Abyss escrow snapshot: %w", err)
	}
	if err := normalizeAbyssSnapshotTables(&snapshot); err != nil {
		return AbyssEscrowSnapshot{}, err
	}
	return snapshot, nil
}
