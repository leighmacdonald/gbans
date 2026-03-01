package main

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	"github.com/leighmacdonald/gbans/internal/cmd"
	_ "golang.org/x/crypto/x509roots/fallback"
)

func main() {
	cmd.Execute()
}
