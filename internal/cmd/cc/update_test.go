package cc

import (
	"testing"

	"github.com/georgealton/rain/cft/format"
	"github.com/georgealton/rain/cft/parse"
	"github.com/georgealton/rain/internal/config"
)

func TestUpdate(t *testing.T) {

	// config.Debug = true

	left, err := parse.File("../../../test/templates/ccdeploy1-state.yaml")
	if err != nil {
		t.Fatal(err)
	}

	right, err := parse.File("../../../test/templates/ccdeploy2.yaml")
	if err != nil {
		t.Fatal(err)
	}

	changes, err := update(left, right)
	if err != nil {
		t.Fatal(err)
	}

	output := format.String(changes, format.Options{
		JSON:     false,
		Unsorted: false,
	})
	config.Debugf("changes: %v", output)

	// TODO - Confirm that the change template resources have the correct State:Action

}
