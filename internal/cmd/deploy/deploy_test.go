package deploy_test

import (
	"testing"

	"github.com/georgealton/rain/cft/parse"
	"github.com/georgealton/rain/cft/pkg"
	"github.com/georgealton/rain/internal/cmd/deploy"
)

func TestHasRainMetadata(t *testing.T) {
	src := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Metadata:
      Rain:
        Something: foo
`
	template, err := parse.String(src)
	if err != nil {
		t.Fatal(err)
	}
	if deploy.HasRainMetadata(template) != true {
		t.Fatal("expected true")
	}

	src = `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Metadata:
      Something: foo
`
	template, err = parse.String(src)
	if err != nil {
		t.Fatal(err)
	}
	template, err = pkg.Template(template, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if deploy.HasRainMetadata(template) {
		t.Fatal("expected false")
	}

}
