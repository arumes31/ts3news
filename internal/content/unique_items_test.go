package content

import (
	"strings"
	"testing"
)

func TestUniqueItemsLogic(t *testing.T) {
	item := RandomUniqueItem()
	if item.Name == "" {
		t.Error("UniqueItem name should not be empty")
	}
	if item.Power <= 0 {
		t.Error("UniqueItem power should be positive")
	}
	
	desc := GetUniqueItemDescription(item)
	if !strings.Contains(desc, item.Name) {
		t.Errorf("Description %q does not contain name %q", desc, item.Name)
	}
}
