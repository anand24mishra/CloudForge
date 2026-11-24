package main

import "testing"

func TestExampleName(t *testing.T) {
	if got := "cloudforge-go-example"; got != "cloudforge-go-example" {
		t.Fatalf("unexpected example name %q", got)
	}
}
