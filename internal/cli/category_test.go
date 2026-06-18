package cli

import (
	"testing"
)

func TestCategoryList(t *testing.T) {
	client, err := NewClient("/tmp/makima-test.sock")
	if err != nil {
		t.Skip("daemon not running")
	}
	defer client.Close()

	categories, err := client.CategoryList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Found %d categories", len(categories))
}
