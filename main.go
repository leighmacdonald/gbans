/*
Copyright © 2020-2023 Leigh MacDonald <leigh.macdonald@gmail.com>
*/
package main

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/leighmacdonald/gbans/internal/cmd"
)

func main() {
	cmd.Execute()
}
