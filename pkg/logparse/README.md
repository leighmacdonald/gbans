# logparse

The logparse library implements a parser for TF2 server logs, transforming them into
statically typed structs.

This includes support for most of the extended output of the following plugins as well:

- SupStats2
- MedicStats

## Match

The additional match functionality will tally up data and give summarized data in a format
that is fairly similar to logs.tf.

```go
package main

import (
 "fmt"
 "github.com/leighmacdonald/gbans/pkg/logparse"
 "github.com/pkg/errors"
)

func main() {
 parser := logparse.New()
 match := logparse.NewMatch(0, "Cool Server")
 logLines := make(chan string)

 go func() {
  // - tail log file from filesystem 
  // - read log file line by line
  // - read logs from a remote source using srcds UDP log listener
  // logLines <- "LOG LINE 1"
  // logLines <- "LOG LINE 2"
 }()

 for {
  line := <-logLines
  if line == "" {
   continue
  }

  result, errResult := parser.Parse(line)
  if errResult != nil {
   panic(errResult)
  }

  if err := match.Apply(result); err != nil && !errors.Is(err, logparse.ErrIgnored) {
   panic(fmt.Sprintf("Failed to Apply: %v [%d] %v", err, result.EventType, line))
  }
 }

}
```
