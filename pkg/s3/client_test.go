package s3

import (
	"context"
	"net/http"
	"testing"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

func TestClient_NewClient_MissingCredentials(t *testing.T) {
	_, err := NewClient()
	if err == nil {
		t.Error("expected error for missing credentials")
	}
}

func TestClient_NewClient_MissingBucket(t *testing.T) {
	_, err := NewClient(
		WithCredentials("access", "secret"),
	)
	if err == nil {
		t.Error("expected error for missing bucket")
	}
}

func TestClient_getObjectKey(t *testing.T) {
	client, err := NewClient(
		WithCredentials("access", "secret"),
		WithBucket("test"),
		WithRegion("us-east-1"),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		time   time.Time
		table  string
		expect string
	}{
		{
			time:   time.Date(2026, 2, 18, 10, 30, 0, 0, time.UTC),
			table:  "temperature",
			expect: "test/temperature/2026/02/18/10.jsonl",
		},
		{
			time:   time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC),
			table:  "humidity",
			expect: "test/humidity/2026/12/31/23.jsonl",
		},
	}

	for _, tc := range tests {
		got := client.getObjectKey(tc.time, tc.table)
		if got != tc.expect {
			t.Errorf("getObjectKey(%v, %s) = %s; want %s", tc.time, tc.table, got, tc.expect)
		}
	}
}

func TestClient_Export_Empty(t *testing.T) {
	client, err := NewClient(
		WithCredentials("access", "secret"),
		WithBucket("test"),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = client.Export(context.Background(), nil)
	if err != nil {
		t.Errorf("Export(nil) = %v; want nil", err)
	}

	err = client.Export(context.Background(), []*api.DataPoint{})
	if err != nil {
		t.Errorf("Export([]) = %v; want nil", err)
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Key: "test-key"}
	if err.Error() != "not found: test-key" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !isNotFound(err) {
		t.Error("expected isNotFound to return true")
	}

	otherErr := &someOtherError{msg: "other"}
	if isNotFound(otherErr) {
		t.Error("expected isNotFound to return false for other error")
	}
}

type someOtherError struct {
	msg string
}

func (e *someOtherError) Error() string {
	return e.msg
}

func TestSigner_SignRequest(t *testing.T) {
	signer := NewSigner("accessKey", "secretKey", "us-east-1")

	req, err := http.NewRequest("GET", "https://s3.us-east-1.redmaple.tree/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "s3.us-east-1.redmaple.tree"

	now := time.Date(2026, 2, 18, 10, 30, 0, 0, time.UTC)
	signer.SignRequest(req, now)

	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate != "20260218T103000Z" {
		t.Errorf("X-Amz-Date = %s; want 20260218T103000Z", amzDate)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Error("Authorization header should not be empty")
	}

	if len(auth) < 50 {
		t.Errorf("Authorization header seems too short: %s", auth)
	}
}
