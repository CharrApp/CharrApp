package parser

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
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
