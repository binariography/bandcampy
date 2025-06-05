package main

import "testing"

func TestRemoveIllegalCharacters(t *testing.T) {
	got := removeIllegalCharacters("This is a / file with slash")
	want := "This is a  file with slash"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
