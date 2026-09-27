package main

import "testing"

func TestRequestRunID(t *testing.T) {
	got, err := requestRunID([]byte(`{"ctx":{"run_id":"01TEST"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != "01TEST" {
		t.Fatalf("run ID = %q, want %q", got, "01TEST")
	}
}

func TestJSONEqualIgnoresObjectKeyOrder(t *testing.T) {
	if !jsonEqual([]byte(`{"left":1,"right":2}`), []byte(`{"right":2,"left":1}`)) {
		t.Fatal("semantically equal JSON did not match")
	}
}
