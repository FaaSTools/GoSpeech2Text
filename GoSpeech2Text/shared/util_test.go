package shared

import (
	"strings"
	"testing"
)

func TestGetFileTypeFromFileName(t *testing.T) {

	result1 := GetFileTypeFromFileName("test.mp3")
	if !strings.EqualFold(result1, "mp3") {
		t.Error("wrong filetype: Got ", result1)
	}

	result2 := GetFileTypeFromFileName("test")
	if !strings.EqualFold(result2, "") {
		t.Error("wrong filetype: Got ", result2)
	}

	result3 := GetFileTypeFromFileName("")
	if !strings.EqualFold(result3, "") {
		t.Error("wrong filetype: Got ", result3)
	}

	result4 := GetFileTypeFromFileName("test.tar.gz")
	if !strings.EqualFold(result4, "gz") {
		t.Error("wrong filetype: Got ", result4)
	}
}
