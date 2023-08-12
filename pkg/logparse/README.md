# logparse

The logparse library implements a parser for TF2 server logs, transforming them into
statically typed structs.

This includes support for most of the extended output of the following plugins as well:

- SupStats2
- MedicStats

## Match

The additional match functionality will tally up data and give summarized data in a format
that is fairly similar to logs.tf.

    match := logparse.NewMatch(logger, serverID, serverName)
    match.Apply(<event>)
