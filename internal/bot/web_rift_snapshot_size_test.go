package bot

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func largeRiftReceipt(t *testing.T) *rift.Run {
	t.Helper()
	run, err := buildLargeRiftReceipt(rift.Build{HP: 300}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func TestRiftFullCampaignLargeReceiptReloads(t *testing.T) {
	run := largeRiftReceipt(t)
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("campaign receipt: %d items, %d JSON bytes", len(run.BankedItems), len(raw))
	encoded, err := encodeRift(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > riftStoredSnapshotLimit || !strings.HasPrefix(encoded, riftCompressedSnapshotPrefix) {
		t.Fatal("large receipt not bounded and compressed")
	}
	t.Logf("stored receipt: %d bytes", len(encoded))
	legacy, err := decodeRift(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := decodeRift(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, legacy) {
		t.Fatal("compressed receipt differs from legacy JSON")
	}
}

func TestRiftSnapshotCodecBoundaries(t *testing.T) {
	run := rift.NewRunAtLevel("small", rift.Build{HP: 300}, time.Unix(100, 0), nil, 1)
	saved, err := encodeRift(run)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(saved, riftCompressedSnapshotPrefix) {
		t.Fatal("small save unexpectedly compressed")
	}
	if _, err = decodeRift(saved); err != nil {
		t.Fatal(err)
	}
	var zipped bytes.Buffer
	zw := gzip.NewWriter(&zipped)
	if _, err = zw.Write(bytes.Repeat([]byte("x"), riftDecodedSnapshotLimit+1)); err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"oversized legacy":   strings.Repeat("x", riftDecodedSnapshotLimit+1),
		"oversized envelope": riftCompressedSnapshotPrefix + strings.Repeat("A", riftStoredSnapshotLimit),
		"base64":             riftCompressedSnapshotPrefix + "!",
		"gzip":               riftCompressedSnapshotPrefix + base64.StdEncoding.EncodeToString([]byte("not gzip")),
		"expansion":          riftCompressedSnapshotPrefix + base64.StdEncoding.EncodeToString(zipped.Bytes()),
		"truncated":          riftCompressedSnapshotPrefix + base64.StdEncoding.EncodeToString(zipped.Bytes()[:len(zipped.Bytes())-5]),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeRift(raw); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
	run.BankedItems = []string{strings.Repeat("x", riftDecodedSnapshotLimit)}
	if _, err := encodeRift(run); err == nil {
		t.Fatal("oversized write accepted")
	}
}

type boundedRiftReceiptArg struct{ items int }

func (a boundedRiftReceiptArg) Match(value driver.Value) bool {
	saved, ok := value.(string)
	if !ok || len(saved) > riftStoredSnapshotLimit {
		return false
	}
	payload, err := riftSnapshotJSON(saved)
	if err != nil {
		return false
	}
	var run rift.Run
	err = json.Unmarshal(payload, &run)
	return err == nil && run.TotalBankedItems() == a.items && len(run.BankedItems) == rift.ReceiptHistoryLimit && len(run.BankedLoot) == rift.ReceiptHistoryLimit && run.Revision == 2
}

func TestRiftLargeLegacyReceiptCompactsOnNextTransaction(t *testing.T) {
	run := largeRiftReceipt(t)
	run.Epoch = "2"
	run.Revision = 1
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", boundedRiftReceiptArg{len(run.BankedItems)}).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	out, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "pause", RunID: run.ID, Revision: 2, RequestID: "compact-existing"}, rift.Build{}, time.Unix(101, 0))
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalBankedItems() != len(run.BankedItems) || len(out.BankedItems) != rift.ReceiptHistoryLimit {
		t.Fatal("transaction lost reward totals or failed to bound receipt")
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftSnapshotRejectsCorruptionAndIncompressibleWrites(t *testing.T) {
	run := largeRiftReceipt(t)
	saved, err := encodeRift(run)
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(saved, riftCompressedSnapshotPrefix))
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range [][]byte{compressed[:len(compressed)-5], append([]byte(nil), compressed...)} {
		if len(payload) == len(compressed) {
			payload[len(payload)-8] ^= 1
		}
		if _, err = decodeRift(riftCompressedSnapshotPrefix + base64.StdEncoding.EncodeToString(payload)); err == nil {
			t.Fatal("corrupted gzip accepted")
		}
	}
	rng := rand.New(rand.NewPCG(1, 2))
	noise := make([]byte, 400000)
	for i := range noise {
		noise[i] = byte('!' + rng.IntN(90))
	}
	run.BankedItems = []string{string(noise)}
	if _, err = encodeRift(run); err == nil {
		t.Fatal("incompressible oversized write accepted")
	}
}
