import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { expect, test } from "vitest";
import { PersonDisplaySchema } from "../../rpc/person/v1/person_core_pb.ts";
import {
	MatchSchema,
	RoundPlayerSchema,
	RoundPlayerVariantSchema,
	RoundSchema,
	Team,
} from "../../rpc/stats/v1/stats_pb.ts";
import { assembleMatch } from "./match.ts";

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
	expect(view.variants[steamId]["tf_projectile_rocket"].kills).toBe(18);
	expect(view.variants[steamId]["tf_projectile_rocket"].isWeapon).toBe(true);
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
	expect(view.variants[demoId]["stickybomb_launcher"].isWeapon).toBe(true);
	expect(view.summaries[0].classes).toContain("demoman");
});
