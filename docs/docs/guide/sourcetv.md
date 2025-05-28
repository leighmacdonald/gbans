# SourceTV

[SourceTV](https://developer.valvesoftware.com/wiki/SourceTV) is a feature provided by the source engine which allows the 
in-engine live broadcasting and recording of games.

gbans provides a few features to support this functionality:

- Automatic downloading (pull) and deletion of demos (.dem) via SSH/SCP
- Scheduled cleanup strategies for removing old demos:
  - Max number of demos
  - Percentage free on disk volume


Some functionality, such as [player stats](stats.md) and pulling steam ids from demos for searching, requires a instance of [tf2_demostats](https://github.com/leighmacdonald/tf2_demostats) to
also be configured.