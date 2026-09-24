package bot

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"ts3news/internal/rift"
)

const riftStoredSnapshotLimit = 256_000
const riftDecodedSnapshotLimit = 2 * 1024 * 1024
const riftCompressedSnapshotPrefix = "rift:gzip:v1:"

// Large receipts keep every item while retaining the existing storage budget.
// Small snapshots stay plain JSON for compatibility and cheap movement updates.
func encodeRift(run *rift.Run) (string, error) {
	raw, err := json.Marshal(run)
	if err != nil {
		return "", err
	}
	if len(raw) > riftDecodedSnapshotLimit {
		return "", errors.New("rift snapshot exceeds decoded limit")
	}
	if len(raw) <= riftStoredSnapshotLimit {
		return string(raw), nil
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(raw); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	saved := riftCompressedSnapshotPrefix + base64.StdEncoding.EncodeToString(compressed.Bytes())
	if len(saved) > riftStoredSnapshotLimit {
		return "", errors.New("rift snapshot exceeds storage limit")
	}
	return saved, nil
}

func riftSnapshotJSON(saved string) ([]byte, error) {
	if !strings.HasPrefix(saved, riftCompressedSnapshotPrefix) {
		// Older writers did not enforce their reader's 256 KB limit. Recover those
		// saves within a bounded decoded budget; the next write uses compression.
		if len(saved) > riftDecodedSnapshotLimit {
			return nil, errors.New("rift snapshot exceeds decoded limit")
		}
		return []byte(saved), nil
	}
	if len(saved) > riftStoredSnapshotLimit {
		return nil, errors.New("rift snapshot exceeds storage limit")
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(saved, riftCompressedSnapshotPrefix))
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(reader, riftDecodedSnapshotLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > riftDecodedSnapshotLimit {
		return nil, errors.New("rift snapshot exceeds decoded limit")
	}
	return raw, nil
}
