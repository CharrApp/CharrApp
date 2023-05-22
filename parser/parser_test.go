package parser

import (
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	_ "embed"
)

//go:embed test.vars
var testContent string

func TestParse(t *testing.T) {
	r := strings.NewReader(testContent)
	config, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() failed: %s", err)
	}
	spew.Dump(config)
}
