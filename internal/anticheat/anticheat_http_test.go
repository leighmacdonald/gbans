package anticheat_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestAnticheat(t *testing.T) {
	var (
		auth      = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)}
		server    = fixture.CreateTestServer(t.Context())
		router    = fixture.CreateRouter()
		antiCheat = anticheat.NewAntiCheat(anticheat.NewRepository(fixture.Database), fixture.Config.Config().Anticheat, notification.NewDiscard(),
			func(_ context.Context, _ logparse.StacEntry, _ time.Duration, _ int) error {
				return nil
			}, fixture.Persons)
	)

	for _, sid := range extra.FindReaderSteamIDs(strings.NewReader(testData)) {
		fixture.CreateTestPerson(t.Context(), sid, permission.User)
	}

	for _, sid := range []string{"STEAM_0:1:59129186", "STEAM_0:0:123751210", "STEAM_0:0:893704961", "STEAM_0:1:807960493"} {
		fixture.CreateTestPerson(t.Context(), steamid.New(sid), permission.User)
	}
	anticheat.NewAnticheatHandler(router, auth, antiCheat)

	entries, err := antiCheat.Import(t.Context(), "stac_052224.log", io.NopCloser(strings.NewReader(testData)), server.ServerID)
	require.NoError(t, err)
	require.Len(t, entries, 9)
}

const testData = `<01:13:00>

----------

           [StAC] SilentAim detection of 1.08° on JSN_.
Detections so far: 1 norecoil = yes
<01:13:00>
Player: JSN_<591><[U:1:118258373]><>
StAC cached SteamID: STEAM_0:1:59129186
<01:13:00>
Network:
60.65 ms ping
0.00 loss
1.33 inchoke
1.20 outchoke
2.54 totalchoke
305.48 kbps rate
133.64 pps rate
<01:13:00>
More network:
Approx client cmdrate: ≈67 cmd/sec
Approx server tickrate: ≈67 tick/sec
Failing lag check? no
SequentialCmdnum? yes
<01:13:00>
Angles:
angles0: x 45.406681 y -85.902275
angles1: x 45.769683 y -84.879280
angles2: x 45.505680 y -85.803283
angles3: x 44.383678 y -89.103279
angles4: x 42.337677 y -94.779273
<01:13:00>
Client eye positions:
eyepos 0: x -523.538 y 1921.645 z 218.336
eyepos 1: x -521.955 y 1924.878 z 215.261
<01:13:00>
Previous cmdnums:
0 146947
1 146946
2 146945
3 146944
4 146943
<01:13:00>
Previous tickcounts:
0 68172
1 68171
2 68170
3 68169
4 68168
<01:13:00>
Current server tick:
68176
<01:13:00>
Mouse Movement (sens weighted):
abs(x): 9
abs(y): 4
Mouse Movement (unweighted):
x: 13
y: -6
Client Sens:
1.500000
<01:13:00>
Previous buttons - use https://sapphonie.github.io/flags.html to convert to readable input
0 527
1 527
2 527
3 527
4 527
<23:15:28>

----------

[StAC] Aimsnap detection of 10.40° on SHADOW.
Detections so far: 1.
<19:37:38>
Player: SHADOW<182><[U:1:247502420]><>
StAC cached SteamID: STEAM_0:0:123751210
<19:37:38>
Network:
92.58 ms ping
0.00 loss
56.24 inchoke
0.00 outchoke
56.24 totalchoke
194.50 kbps rate
61.34 pps rate
<19:37:38>
More network:
Approx client cmdrate: ≈69 cmd/sec
Approx server tickrate: ≈67 tick/sec
Failing lag check? no
SequentialCmdnum? yes
<19:37:38>
Angles:
angles0: x 9.236021 y 66.065795
angles1: x 8.893921 y 65.860534
angles2: x 10.123413 y 76.196166
angles3: x 10.260252 y 76.264587
angles4: x 10.328673 y 76.333007
<19:37:38>
Client eye positions:
eyepos 0: x -419.240 y 20.790 z 90.161
eyepos 1: x -420.066 y 16.244 z 90.161
<19:37:38>
Previous cmdnums:
0 345380
1 345379
2 345378
3 345377
4 345376
<19:37:38>
Previous tickcounts:
0 23191
1 23190
2 23189
3 23188
4 23187
<19:37:38>
Current server tick:
23198
<19:37:38>
Mouse Movement (sens weighted):
abs(x): 2
abs(y): 4
Mouse Movement (unweighted):
x: -6
y: 12
Client Sens:
3.109999
<19:37:38>
Previous buttons - use https://sapphonie.github.io/flags.html to convert to readable input
0 9
1 9
2 9
3 9
4 9
<19:37:38>
Angle deltas:
0 0.398954
1 10.408503
2 0.152991
3 0.096761

<19:37:38>

----------

[StAC] Cmdnum SPIKE of 238 on MashingButtons.
Detections so far: 1.
<15:39:55>
Player: MashingButtons<26><[U:1:1787409922]><>
StAC cached SteamID: STEAM_0:0:893704961
<15:39:55>
Network:
60.12 ms ping
0.01 loss
1.30 inchoke
0.00 outchoke
1.30 totalchoke
80.56 kbps rate
134.01 pps rate
<15:39:55>
More network:
Approx client cmdrate: ≈67 cmd/sec
Approx server tickrate: ≈67 tick/sec
Failing lag check? yes
SequentialCmdnum? no
<15:39:55>
Previous cmdnums:
0 6403
1 6165
2 6165
3 6165
4 6165
<15:39:55>
Previous tickcounts:
0 31295
1 31546
2 31545
3 31544
4 31543
<15:39:55>
Current server tick:
31782
<15:39:55> Held weapon: tf_weapon_rocketlauncher
<17:35:14>

----------

[StAC] Player dong bhopped!
Consecutive detections so far: 21
<17:39:39>
Player: dong<7><[U:1:1847123999]><>
StAC cached SteamID: STEAM_0:1:923561999
<17:39:40>

----------

[StAC] Player aeditorkid3 has invalid eye angles!
Current angles: 90.00 -56.70 0.00.
Detections so far: 1
<08:41:52>
Player: aeditorkid3<70><[U:1:1843380313]><>
StAC cached SteamID: STEAM_0:1:921690156
<08:41:52>
Network:
35.52 ms ping
1.19 loss
0.44 inchoke
0.72 outchoke
1.16 totalchoke
256.71 kbps rate
133.40 pps rate
<08:41:52>
More network:
Approx client cmdrate: ≈67 cmd/sec
Approx server tickrate: ≈67 tick/sec
Failing lag check? yes
SequentialCmdnum? yes
<08:41:52>
Angles:
angles0: x 90.000000 y -56.707004
angles1: x 16.421136 y 33.421787
angles2: x 16.421136 y 33.486183
angles3: x 16.421136 y 33.486183
angles4: x 16.421136 y 33.486183
<08:41:52>
Client eye positions:
eyepos 0: x 2309.573 y -1935.004 z -134.348
eyepos 1: x 2304.742 y -1937.724 z -127.508
<08:41:52>
Previous cmdnums:
0 697
1 696
2 695
3 694
4 693
<08:41:52>
Previous tickcounts:
0 1649
1 1648
2 1647
3 1646
4 1645
<08:41:52>
Current server tick:
1653
<08:41:59>

----------

[StAC] Player (2)DoesVac sent an invalid usercmd!
               Buttons 134217728 were invalid, >= (1 << 26)!.
Detections so far: 1
<05:28:30>
Player: (2)DoesVac<970><[U:1:1685705304]><>
StAC cached SteamID: STEAM_0:0:842852652
<05:28:30>
Network:
238.05 ms ping
0.00 loss
0.00 inchoke
0.00 outchoke
0.00 totalchoke
247.84 kbps rate
101.28 pps rate
<05:28:30>
More network:
Approx client cmdrate: ≈67 cmd/sec
Approx server tickrate: ≈67 tick/sec
Failing lag check? no
SequentialCmdnum? yes
<05:28:30>
Angles:
angles0: x 0.000000 y 45.999755
angles1: x 0.000000 y 45.999755
angles2: x 0.000000 y 45.999755
angles3: x 0.000000 y 45.999755
angles4: x 0.000000 y 45.999755
<05:28:30>
Client eye positions:
eyepos 0: x -2368.000 y -1184.000 z 331.031
eyepos 1: x -2368.000 y -1184.000 z 331.031
<05:28:30>
Previous cmdnums:
0 1289
1 1288
2 1287
3 1286
4 1285
<05:28:30>
Previous tickcounts:
0 23946
1 23945
2 23944
3 23943
4 23942
<05:28:30>
Current server tick:
23961
<05:28:30>
Mouse Movement (sens weighted):
abs(x): 0
abs(y): 0
Mouse Movement (unweighted):
x: 0
y: 0
Client Sens:
0.000000
<05:28:30>
Previous buttons - use https://sapphonie.github.io/flags.html to convert to readable input
0 134217728
1 0
2 134217728
3 0
4 134217728
<05:28:30> Demo file: stv_demos/active/20240512-052232-pl_badwater.dem. Demo tick: 23861
<05:28:30>

----------

[StAC] [Detection] Player Miner mol lester is cheating - OOB cvar/netvar value -1 on var cl_cmdrate!
<21:04:24>
Player: Miner mol lester<240><[U:1:429192328]><>
StAC cached SteamID: STEAM_0:0:214596164
<21:04:24> Demo file: stv_demos/active/20241202-203341-cp_mountainlab.dem. Demo tick: 122871

----------

[StAC] [Detection] Player The Bread Baker is cheating - detected known cheat var/concommand windows_speaker_config!
<17:49:49>
Player: The Bread Baker<88><[U:1:1615920987]><>
StAC cached SteamID: STEAM_0:1:807960493
<17:49:49> Demo file: stv_demos/active/20240522-174456-pl_upward.dem. Demo tick: 19525
<17:57:24>

----------

[StAC] [Detection] Player The Bread Baker is cheating - detected known cheat var/concommand windows_speaker_config!
<17:49:49>
Player: The Bread Baker<88><XXXX><>
StAC cached SteamID: STEAM_0:1:807960493
<17:49:49> Demo file: stv_demos/active/20240522-174456-pl_upward.dem. Demo tick: 19525
<17:57:24>
`
