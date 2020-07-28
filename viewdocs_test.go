package main

import (
	"fmt"
	"testing"
)

func TestInvalidEnvOnDisk(t *testing.T) {
	var f frontmatter

	templateName := "potato"
	body := "<h1>street</h1>"

	contents := fmt.Sprintf("---\nTemplateName: %s\n---\n%s", templateName, body)

	str, err := parseFrontmatter(contents, &f)

	if err != nil {
		t.Errorf("Error parsing frontmatter: %v", err)
	}

	if f.TemplateName != templateName {
		t.Errorf("Expected TemplateName to be %q, got %q", templateName, f.TemplateName)
	}

	if str != body {
		t.Errorf("Expected TemplateName to be %q, got %q", body, str)
	}
}
