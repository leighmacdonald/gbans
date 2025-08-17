//go:generate go tool oapi-codegen -config .openapi.yaml https://tf-api.roto.lol/api/openapi/schema-3.0.json
package main

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/leighmacdonald/gbans/internal/cmd"
)

func main() {
	cmd.Execute()
}
