package nestedvirt

import "testing"

func TestParseLibvirtUUID(t *testing.T) {
	const value = "11112222-3333-4444-5555-666677778888"

	uuid, err := parseLibvirtUUID(value)
	if err != nil {
		t.Fatal(err)
	}

	if got := formatLibvirtUUID(uuid); got != value {
		t.Fatalf("formatLibvirtUUID() = %q, want %q", got, value)
	}
}

func TestParseLibvirtUUIDRejectsInvalidLength(t *testing.T) {
	if _, err := parseLibvirtUUID("11112222-3333-4444-5555"); err == nil {
		t.Fatal("parseLibvirtUUID() error = nil, want error")
	}
}
