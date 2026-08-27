package db

import (
	"bytes"
	"strings"
	"testing"
)

func TestAbyssEscrowSnapshotEncodeDecode(t *testing.T) {
	t.Parallel()

	snapshot := testAbyssEscrowSnapshot(t)
	var encoded bytes.Buffer
	if err := EncodeAbyssEscrowSnapshot(&encoded, snapshot); err != nil {
		t.Fatalf("EncodeAbyssEscrowSnapshot: %v", err)
	}
	decoded, err := DecodeAbyssEscrowSnapshot(bytes.NewReader(encoded.Bytes()), int64(encoded.Len()))
	if err != nil {
		t.Fatalf("DecodeAbyssEscrowSnapshot: %v\n%s", err, encoded.String())
	}
	if decoded.Checksum != snapshot.Checksum || decoded.Counts != snapshot.Counts {
		t.Fatalf("decoded metadata = checksum %q counts %+v", decoded.Checksum, decoded.Counts)
	}
}

func TestDecodeAbyssEscrowSnapshotBoundsAndChecksum(t *testing.T) {
	t.Parallel()

	snapshot := testAbyssEscrowSnapshot(t)
	var encoded bytes.Buffer
	if err := EncodeAbyssEscrowSnapshot(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		input   []byte
		limit   int64
		wantErr string
	}{
		{name: "too large", input: encoded.Bytes(), limit: int64(encoded.Len() - 1), wantErr: "exceeds"},
		{name: "tampered", input: bytes.Replace(encoded.Bytes(), []byte(`"amount": 5`), []byte(`"amount": 6`), 1), limit: int64(encoded.Len()), wantErr: "checksum"},
		{name: "invalid limit", input: encoded.Bytes(), limit: 0, wantErr: "byte limit"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeAbyssEscrowSnapshot(bytes.NewReader(test.input), test.limit)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("DecodeAbyssEscrowSnapshot error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}
