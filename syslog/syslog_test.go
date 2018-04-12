package syslog

import "testing"

func TestFacilityPriority(t *testing.T) {
	for facility, priority := range facilityMap {
		actual, err := FacilityPriority(facility)
		if err != nil {
			t.Fatalf("Should not return error on valid facility: %s", facility)
		}

		if actual != priority {
			t.Fatalf("Expected returned priority: %d, actual: %d", priority, actual)
		}
	}

	_, err := FacilityPriority("<UNKNOWN>")
	if err == nil {
		t.Fatalf("For invalid facilities, FacilityPriority() should returns error")
	}
}
