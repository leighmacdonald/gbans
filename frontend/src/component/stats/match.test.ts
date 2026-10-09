import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { expect, test } from "vitest";
import { PersonDisplaySchema } from "../../rpc/person/v1/person_core_pb.ts";
import {
	MatchEventSchema,
	MatchKillSchema,
	MatchSchema,
	RoundPlayerSchema,
	RoundPlayerVariantSchema,
	RoundSchema,
	Team,
} from "../../rpc/stats/v1/stats_pb.ts";
import { assembleMatch, eventMapLayerKey, eventMapLayerLabel } from "./match.ts";

const steamId = "76561197960287930";

const variant = (name: string, kills: number) =>
	create(RoundPlayerVariantSchema, { variant: name, kills: String(kills) });

const player = (kills: number, variants: ReturnType<typeof variant>) =>
	create(RoundPlayerSchema, {
		person: create(PersonDisplaySchema, { steamId, name: "shnowshner" }),
		team: Team.BLU,
		kills: String(kills),
		variants,
	});

const round = (players: ReturnType<typeof player>, durationMs = "300000") =>
	create(RoundSchema, { winner: Team.BLU, durationMs, players });

test("variant stats accumulate across rounds", () => {
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: { [steamId]: create(PersonDisplaySchema, { steamId, name: "shnowshner" }) },
		rounds: [
			round([player(10, [variant("soldier", 10), variant("tf_projectile_rocket", 7)])]),
			round([player(18, [variant("soldier", 18), variant("tf_projectile_rocket", 11)])]),
			// Zero-duration stub round: must be ignored like the summary loop does.
			round([player(0, [variant("soldier", 99), variant("spy", 5)])], "0"),
		],
	});

	const view = assembleMatch(data);

	expect(view.summaries).toHaveLength(1);
	expect(view.summaries[0].kills).toBe(28);
	expect(view.variants[steamId].soldier.kills).toBe(28);
	expect(view.variants[steamId].soldier.isWeapon).toBe(false);
	expect(view.variants[steamId].tf_projectile_rocket.kills).toBe(18);
	expect(view.variants[steamId].tf_projectile_rocket.isWeapon).toBe(true);
	expect(view.summaries[0].classes).toContain("soldier");
	expect(view.variants[steamId]).not.toHaveProperty("spy");
});

test("demoman is classified as a class, not a weapon", () => {
	const demoId = "76561198000000001";
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: { [demoId]: create(PersonDisplaySchema, { steamId: demoId, name: "demo" }) },
		rounds: [
			create(RoundSchema, {
				winner: Team.BLU,
				durationMs: "300000",
				players: [
					create(RoundPlayerSchema, {
						person: create(PersonDisplaySchema, { steamId: demoId, name: "demo" }),
						team: Team.BLU,
						kills: "8",
						variants: [variant("demoman", 6), variant("stickybomb_launcher", 2)],
					}),
				],
			}),
		],
	});

	const view = assembleMatch(data);

	expect(view.summaries).toHaveLength(1);
	expect(view.summaries[0].kills).toBe(8);
	expect(view.variants[demoId].demoman.isWeapon).toBe(false);
	expect(view.variants[demoId].demoman.kills).toBe(6);
	expect(view.variants[demoId].stickybomb_launcher.isWeapon).toBe(true);
	expect(view.summaries[0].classes).toContain("demoman");
});

test("kill feed keeps killer/victim positions for the overview map", () => {
	const victimId = "76561198000000002";
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: { [steamId]: create(PersonDisplaySchema, { steamId, name: "shnowshner" }) },
		rounds: [round([player(1, [variant("soldier", 1)])])],
		kills: [
			create(MatchKillSchema, {
				matchKillId: "42",
				tick: 1234,
				killerSteamId: steamId,
				victimSteamId: victimId,
				weapon: "tf_projectile_rocket",
				killerPosX: 100.5,
				killerPosY: -200.25,
				victimPosX: 150.75,
				victimPosY: -250.5,
			}),
		],
	});

	const view = assembleMatch(data);

	expect(view.kills).toHaveLength(1);
	expect(view.kills[0].matchKillId).toBe("42");
	expect(view.kills[0].killerX).toBe(100.5);
	expect(view.kills[0].killerY).toBe(-200.25);
	expect(view.kills[0].victimX).toBe(150.75);
	expect(view.kills[0].victimY).toBe(-250.5);
	expect(view.kills[0].weapon).toBe("tf_projectile_rocket");
	expect(view.kills[0].ticksSinceStart).toBe(1234);
});

test("event feed lists every stored event with summaries and rounds", () => {
	const victimId = "76561198000000002";
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: {
			[steamId]: create(PersonDisplaySchema, { steamId, name: "shnowshner" }),
			[victimId]: create(PersonDisplaySchema, { steamId: victimId, name: "victim" }),
		},
		rounds: [round([player(1, [variant("soldier", 1)])])],
		events: [
			create(MatchEventSchema, {
				matchEventId: "7",
				tick: 100,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "sentry",
				details: JSON.stringify({ owner: steamId, building: "sentry", level: 3, pos: { x: 1, y: 2, z: 3 } }),
			}),
			create(MatchEventSchema, {
				matchEventId: "8",
				tick: 200,
				eventType: "kill",
				actorSteamId: steamId,
				targetSteamId: victimId,
				weapon: "scattergun",
				details: JSON.stringify({ killer: steamId, victim: victimId, weapon: "scattergun" }),
			}),
			create(MatchEventSchema, {
				matchEventId: "9",
				tick: 300,
				eventType: "setup_finished",
				details: "{}",
			}),
		],
	});

	const view = assembleMatch(data);

	expect(view.events).toHaveLength(3);
	expect(view.events[0].eventType).toBe("building_built");
	expect(view.events[0].summary).toBe("shnowshner built a level 3 sentry");
	expect(view.events[1].eventType).toBe("kill");
	expect(view.events[1].summary).toBe("shnowshner killed victim with scattergun");
	expect(view.events[2].summary).toBe("Setup finished");
	// The fixture round has no tick window, so events keep a zero round and
	// ticks measured from the start of the match.
	expect(view.events[0].round).toBe(0);
	expect(view.events[0].ticksSinceStart).toBe(100);
});

test("event feed carries map position, tick, round and team for the event map", () => {
	const victimId = "76561198000000002";
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: {
			[steamId]: create(PersonDisplaySchema, { steamId, name: "shnowshner" }),
			[victimId]: create(PersonDisplaySchema, { steamId: victimId, name: "victim" }),
		},
		rounds: [round([player(1, [variant("soldier", 1)])])],
		events: [
			create(MatchEventSchema, {
				matchEventId: "7",
				tick: 100,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "sentry",
				details: JSON.stringify({ owner: steamId, building: "sentry", level: 3, pos: { x: 1, y: 2, z: 3 } }),
			}),
			create(MatchEventSchema, {
				matchEventId: "8",
				tick: 200,
				eventType: "kill",
				actorSteamId: steamId,
				targetSteamId: victimId,
				weapon: "scattergun",
				details: JSON.stringify({
					killer: steamId,
					victim: victimId,
					weapon: "scattergun",
					killer_pos: { x: 10, y: 20, z: 30 },
					victim_pos: { x: 11, y: 21, z: 31 },
				}),
			}),
			create(MatchEventSchema, {
				matchEventId: "9",
				tick: 300,
				eventType: "setup_finished",
				details: "{}",
			}),
		],
	});

	const view = assembleMatch(data);

	// Building event keeps its `pos` payload with tick/round/team for filtering.
	expect(view.events[0].eventX).toBe(1);
	expect(view.events[0].eventY).toBe(2);
	expect(view.events[0].eventZ).toBe(3);
	expect(view.events[0].tick).toBe(100);
	expect(view.events[0].team).toBe(Team.BLU);
	// Kill events prefer the victim (death) location and keep the attacker end for lines.
	expect(view.events[1].eventX).toBe(11);
	expect(view.events[1].eventY).toBe(21);
	expect(view.events[1].eventZ).toBe(31);
	expect(view.events[1].attackerX).toBe(10);
	expect(view.events[1].attackerY).toBe(20);
	expect(view.events[1].attackerZ).toBe(30);
	expect(view.events[1].team).toBe(Team.BLU);
	// Buildings have no attacker end.
	expect(view.events[0].attackerX).toBeNull();
	// Non-positional events are excluded from map layers via null coordinates.
	expect(view.events[2].eventX).toBeNull();
	expect(view.events[2].eventY).toBeNull();
});

test("building_built map layers split by building kind and skip mini sentries", () => {
	const data = create(MatchSchema, {
		overview: {
			duration: "1260",
			hostname: "test",
			scoreBlu: 1,
			scoreRed: 1,
			createdOn: create(TimestampSchema, { seconds: 1725668520n, nanos: 0 }),
		},
		players: { [steamId]: create(PersonDisplaySchema, { steamId, name: "shnowshner" }) },
		rounds: [round([player(1, [variant("soldier", 1)])])],
		events: [
			create(MatchEventSchema, {
				matchEventId: "10",
				tick: 100,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "sentry",
				details: JSON.stringify({ owner: steamId, building: "sentry", level: 3, pos: { x: 1, y: 2, z: 3 } }),
			}),
			create(MatchEventSchema, {
				matchEventId: "11",
				tick: 200,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "dispenser",
				details: JSON.stringify({ owner: steamId, building: "dispenser", pos: { x: 4, y: 5, z: 6 } }),
			}),
			create(MatchEventSchema, {
				matchEventId: "12",
				tick: 300,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "teleporter",
				details: JSON.stringify({ owner: steamId, building: "teleporter", pos: { x: 7, y: 8, z: 9 } }),
			}),
			create(MatchEventSchema, {
				matchEventId: "13",
				tick: 400,
				eventType: "building_built",
				actorSteamId: steamId,
				building: "sentry",
				details: JSON.stringify({
					owner: steamId,
					building: "sentry",
					level: 1,
					is_mini: true,
					pos: { x: 10, y: 11, z: 12 },
				}),
			}),
		],
	});

	const view = assembleMatch(data);

	expect(view.events[0].isMini).toBe(false);
	expect(view.events[3].isMini).toBe(true);
	expect(eventMapLayerKey(view.events[0])).toBe("building_built_sentry");
	expect(eventMapLayerKey(view.events[1])).toBe("building_built_dispenser");
	expect(eventMapLayerKey(view.events[2])).toBe("building_built_teleporter");
	// Mini/gunslinger sentries get no map layer.
	expect(eventMapLayerKey(view.events[3])).toBeNull();
	expect(eventMapLayerLabel("building_built_sentry")).toBe("Sentries");
	expect(eventMapLayerLabel("building_built_dispenser")).toBe("Dispensers");
	expect(eventMapLayerLabel("building_built_teleporter")).toBe("Teleporters");
});
