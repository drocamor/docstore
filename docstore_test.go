package docstore

import (
	"testing"
)

func TestValidateDocId(t *testing.T) {
	goodIds := []string{
		"foo",
		"bar123",
		"123foo",
		"1-2-bod",
		"foo_bar",
	}

	badIds := []string{
		"Foo",
		"a doc id",
		"foo/bar",
		`123
foo`,
	}

	for _, id := range goodIds {
		err := ValidateDocId(id)
		if err != nil {
			t.Errorf("id %q should be valid, but it was marked invalid", id)
		}
	}

	for _, id := range badIds {
		err := ValidateDocId(id)
		if err == nil {
			t.Errorf("id %q should be invalid, but it was marked valid", id)
		}
	}
}
