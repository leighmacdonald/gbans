# Stats Overview

Stats are generated for many game events. They are summarized into atomic matches and stored in the database. The
design goal is to have a cross between [logs.tf](https://logs.tf)
and [hlstatsx:ce](https://github.com/A1mDev/hlstatsx-community-edition).

## Compared with hlstatsx:ce

- Considerably better scaling performance:
  - Matches are committed to the database in a single transaction instead of immediately upon incoming events
  - Demo parser is written in rust and generally quite fast

:::note

TODO

:::

:::info

At some point in the future we plan to introduce a special "stats only" mode of gbans which is designed to
effectively act as a replacement for hlstatx, without all the extra community aspects enabled. Even with our incomplete
stats system, we had many operators asking if this would be a potential option.

:::
