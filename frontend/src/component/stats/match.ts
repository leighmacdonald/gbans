import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import type { PersonDisplay } from "../../rpc/person/v1/person_core_pb";
import {
	type Match,
	type MatchChatLog,
	type MatchKill,
	type RoundPlayer,
	type RoundPlayerVariant,
	Team,
} from "../../rpc/stats/v1/stats_pb";
import { classList } from "../../tf2";
import { emptyOrNullString } from "../../util/types";

export type MatchRow = {
	player: PersonDisplay;
	team: Team;
	classes: string[];
	points: number;
	kills: number;
	assists: number;
	deaths: number;
	damage: number;
	damagePerMin: number;
	kad: number;
	kd: number;
	kaPerD: number;
	dt: number;
	dtPerMin: number;
	// dtm: number;
	hp: number;
	heals: number;
	charges: number;
	chargesUber: number;
	chargesKritz: number;
	chargesVacc: number;
	chargesQuickfix: number;
	as: number;
	bs: number;
	bsk: number;
	hs: number;
	hsk: number;
	wasHs: number;
	wasBs: number;
	cap: number;
	capturesBlocked: number;
	healing: number;
	drops: number;
	shots: number;
	hits: number;
};

type MatchInfo = {
	hostname: string;
	scoreRed: number;
	scoreBlu: number;
	duration: number;
	mapId: number;
	mapName: string;
	createdOn: Date;
};

export type MatchRound = {
	round: number;
	winner: Team;
	durationMs: number;
	isStalemate: boolean;
	isSuddenDeath: boolean;
	players: RoundPlayer[];
	// logs.tf-style per-round aggregates
	tickStart: number;
	tickEnd: number;
	scoreRed: number;
	scoreBlu: number;
	killsRed: number;
	killsBlu: number;
	damageRed: number;
	damageBlu: number;
	chargesRed: number;
	chargesBlu: number;
};

export type MatchPlayerVariantStats = {
	revenges: number;
	revenged: number;
	drops: number;
	nearFullChargeDeath: number;
	airshots: number;
	backstabs: number;
	backstabKills: number;
	headshots: number;
	headshotKills: number;
	dominations: number;
	dominated: number;
	damage: number;
	damageTaken: number;
	chargesUber: number;
	chargesKritz: number;
	chargesVacc: number;
	chargesQuickfix: number;
	name: string;
	isWeapon: boolean;
	kills: number;
	assists: number;
	deaths: number;
	healing: number;
};

export type TeamTotals = {
	team: Team;
	players: number;
	kills: number;
	assists: number;
	deaths: number;
	damage: number;
	damageTaken: number;
	healing: number;
	charges: number;
	drops: number;
	captures: number;
	capturesBlocked: number;
};

export type KillFeedEntry = {
	tick: number;
	round: number;
	killerSteamId: string;
	victimSteamId: string;
	weapon: string;
};

export type ChatFeedEntry = {
	tick: number;
	round: number;
	steamId: string;
	name: string;
	body: string;
	/** Ticks since the start of the match, derived from demo ticks. */
	ticksSinceStart: number;
};

export type MatchView = {
	info: MatchInfo;
	summaries: MatchRow[];
	rounds: MatchRound[];
	chat: MatchChatLog[];
	variants: Record<string, Record<string, MatchPlayerVariantStats>>;
	durationMins: number;
	teamTotals: TeamTotals[];
	kills: KillFeedEntry[];
	chatFeed: ChatFeedEntry[];
	players: Record<string, PersonDisplay>;
};

export const assembleMatch = (data: Match): MatchView => {
	if (!data.overview) {
		throw "invalid overview";
	}
	const summaries: Record<string, MatchRow> = {};
	//const players = data.players;
	const rounds: MatchRound[] = [];
	const durationSecs = Math.max(1, Number(data.overview.duration));
	const durationMins = durationSecs / 60;

	let scoreRed = 0;
	let scoreBlu = 0;
	for (let i = 0; i < data.rounds.length; i++) {
		if (Number(data.rounds[i].durationMs) === 0) {
			continue;
		}
		const winner = data.rounds[i].winner;
		if (winner === Team.RED) {
			scoreRed += 1;
		} else if (winner === Team.BLU) {
			scoreBlu += 1;
		}
		const agg = aggregateRoundPlayers(data.rounds[i].players);
		rounds.push({
			round: i + 1,
			winner,
			durationMs: Number(data.rounds[i].durationMs),
			isStalemate: data.rounds[i].isStalemate,
			isSuddenDeath: data.rounds[i].isSuddenDeath,
			players: data.rounds[i].players,
			tickStart: agg.tickStart,
			tickEnd: agg.tickEnd,
			scoreRed,
			scoreBlu,
			killsRed: agg.killsRed,
			killsBlu: agg.killsBlu,
			damageRed: agg.damageRed,
			damageBlu: agg.damageBlu,
			chargesRed: agg.chargesRed,
			chargesBlu: agg.chargesBlu,
		});
		for (let p = 0; p < data.rounds[i].players.length; p++) {
			const steamId = data?.rounds[i].players[p].person?.steamId ?? "";
			if (emptyOrNullString(steamId)) {
				continue;
			}

			if (!Object.hasOwn(summaries, steamId)) {
				const po = data.players[String(steamId)];
				if (!po) {
					continue;
				}
				summaries[steamId] = newMatchRow(po);
			}
			const rp = data.rounds[i].players[p];
			const sm = summaries[steamId];

			sm.bsk += Number(rp.backstabKills);
			sm.drops += Number(rp.drops);
			sm.hits += Number(rp.hits);
			sm.hsk += Number(rp.headshotKills);
			sm.points += Number(rp.points);
			sm.healing += Number(rp.healing);
			sm.shots += Number(rp.shots);
			sm.wasBs += Number(rp.wasBackstabbed);
			sm.wasHs += Number(rp.wasHeadshot);
			sm.as += Number(rp.airshots);
			sm.hs += Number(rp.headshots);
			sm.bs += Number(rp.backstabs);
			sm.assists += Number(rp.assists);
			sm.cap += Number(rp.captures);
			sm.capturesBlocked += Number(rp.capturesBlocked);
			sm.damage += Number(rp.scoreboardDamage);
			sm.deaths += Number(rp.scoreboardDeaths);
			sm.dt += Number(rp.damageTaken);
			sm.kills += Number(rp.kills);
			sm.heals += Number(rp.heals);
			sm.chargesUber += Number(rp.chargesUber);
			sm.chargesKritz += Number(rp.chargesKritz);
			sm.chargesVacc += Number(rp.chargesVacc);
			sm.chargesQuickfix += Number(rp.chargesQuickfix);
			sm.charges +=
				Number(rp.chargesUber) + Number(rp.chargesKritz) + Number(rp.chargesVacc) + Number(rp.chargesQuickfix);
			// summaries[data.rounds[i].players[p].steamId].name = data.rounds[i].players[p].;
			sm.team = rp.team;
		}
	}

	const variants = assemblePlayerVariants(data);

	const summaryList = Object.values(summaries).map((sm) => {
		// Classes played, derived from per-variant stats (non-weapon variants are classes).
		const playerVariants = variants[sm.player.steamId];
		if (playerVariants) {
			sm.classes = Object.values(playerVariants)
				.filter((v) => !v.isWeapon)
				.map((v) => v.name)
				.toSorted();
		}
		sm.damagePerMin = sm.damage / durationMins;
		sm.dtPerMin = sm.dt / durationMins;
		sm.kd = sm.kills / Math.max(1, sm.deaths);
		sm.kaPerD = (sm.kills + sm.assists) / Math.max(1, sm.deaths);
		// Legacy alias kept for compat.
		sm.kad = sm.kaPerD;
		return sm;
	});

	return {
		info: {
			createdOn: timestampDate(data.overview.createdOn as Timestamp),
			duration: Number(data.overview.duration),
			hostname: data.overview.hostname,
			mapName: data.overview.map?.name as string,
			mapId: data.overview.map?.mapId as number,
			scoreRed: data.overview.scoreRed,
			scoreBlu: data.overview.scoreBlu,
		},
		rounds,
		summaries: summaryList.toSorted((a, b) => b.points - a.points),
		chat: data.chatLogs,
		variants,
		durationMins,
		teamTotals: aggregateTeamTotals(summaryList),
		kills: assembleKillFeed(data.kills, rounds),
		chatFeed: assembleChatFeed(data.chatLogs, rounds),
		players: data.players ?? {},
	};
};

const assemblePlayerVariants = (data: Match): Record<string, Record<string, MatchPlayerVariantStats>> => {
	const variantSummaries: Record<string, Record<string, MatchPlayerVariantStats>> = {};
	for (let r = 0; r < data.rounds.length; r++) {
		for (let p = 0; p < data.rounds[r].players.length; p++) {
			const steamId = data?.rounds[r].players[p].person?.steamId ?? "";
			if (emptyOrNullString(steamId)) {
				continue;
			}

			for (let v = 0; v < data?.rounds[r].players[p].variants.length; v++) {
				if (!Object.hasOwn(variantSummaries, steamId)) {
					variantSummaries[steamId] = {};
				}
				const variantStats = data?.rounds[r].players[p].variants[v];
				const isWeapon = !classList.includes(variantStats.variant);
				if (!Object.hasOwn(variantSummaries, variantStats.variant)) {
					variantSummaries[steamId][variantStats.variant] = newVariant(variantStats);
				}
				const variant = variantSummaries[steamId][variantStats.variant];
				variant.kills += Number(variantStats.kills);
				variant.assists += Number(variantStats.assists);
				variant.deaths += Number(variantStats.deaths);
				variant.healing += Number(variantStats.healing);
				variant.airshots += Number(variantStats.airshots);
				variant.backstabs += Number(variantStats.backstabs);
				variant.backstabKills += Number(variantStats.backstabKills);
				variant.headshots += Number(variantStats.headshots);
				variant.headshotKills += Number(variantStats.headshotKills);
				variant.dominations += Number(variantStats.dominations);
				variant.dominated += Number(variantStats.dominated);
				variant.damage += Number(variantStats.damage);
				variant.revenges += Number(variantStats.revenges);
				variant.revenged += Number(variantStats.revenged);
				variant.damageTaken += Number(variantStats.damageTaken);
				variant.chargesUber += Number(variantStats.chargesUber);
				variant.chargesKritz += Number(variantStats.chargesKritz);
				variant.chargesVacc += Number(variantStats.chargesVacc);
				variant.chargesQuickfix += Number(variantStats.chargesQuickfix);
				variant.drops += Number(variantStats.drops);
				variant.nearFullChargeDeath += Number(variantStats.nearFullChargeDeath);
				variant.isWeapon = isWeapon;
			}
		}
	}

	return variantSummaries;
};

const newMatchRow = (po: PersonDisplay): MatchRow => ({
	bsk: 0,
	drops: 0,
	hits: 0,
	hsk: 0,
	points: 0,
	shots: 0,
	wasBs: 0,
	wasHs: 0,
	as: 0,
	bs: 0,
	assists: 0,
	cap: 0,
	classes: [],
	damage: 0,
	damagePerMin: 0,
	deaths: 0,
	dt: 0,
	dtPerMin: 0,
	// dtm: 0,
	hp: 0,
	heals: 0,
	charges: 0,
	chargesUber: 0,
	chargesKritz: 0,
	chargesVacc: 0,
	chargesQuickfix: 0,
	kad: 0,
	kaPerD: 0,
	kd: 0,
	kills: 0,
	hs: 0,
	player: po,
	healing: 0,
	capturesBlocked: 0,
	team: Team.UNASSIGNED_UNSPECIFIED,
});

const aggregateRoundPlayers = (players: RoundPlayer[]) => {
	let tickStart = Number.MAX_SAFE_INTEGER;
	let tickEnd = 0;
	let killsRed = 0;
	let killsBlu = 0;
	let damageRed = 0;
	let damageBlu = 0;
	let chargesRed = 0;
	let chargesBlu = 0;
	for (const p of players) {
		const start = Number(p.tickStart);
		const end = Number(p.tickEnd);
		if (Number.isFinite(start) && start < tickStart) {
			tickStart = start;
		}
		if (Number.isFinite(end) && end > tickEnd) {
			tickEnd = end;
		}
		const charges =
			Number(p.chargesUber) + Number(p.chargesKritz) + Number(p.chargesVacc) + Number(p.chargesQuickfix);
		if (p.team === Team.RED) {
			killsRed += Number(p.kills);
			damageRed += Number(p.scoreboardDamage);
			chargesRed += charges;
		} else if (p.team === Team.BLU) {
			killsBlu += Number(p.kills);
			damageBlu += Number(p.scoreboardDamage);
			chargesBlu += charges;
		}
	}
	return {
		tickStart: tickStart === Number.MAX_SAFE_INTEGER ? 0 : tickStart,
		tickEnd,
		killsRed,
		killsBlu,
		damageRed,
		damageBlu,
		chargesRed,
		chargesBlu,
	};
};

const newTeamTotals = (team: Team): TeamTotals => ({
	team,
	players: 0,
	kills: 0,
	assists: 0,
	deaths: 0,
	damage: 0,
	damageTaken: 0,
	healing: 0,
	charges: 0,
	drops: 0,
	captures: 0,
	capturesBlocked: 0,
});

const aggregateTeamTotals = (summaries: MatchRow[]): TeamTotals[] => {
	const totals: Record<number, TeamTotals> = {
		[Team.RED]: newTeamTotals(Team.RED),
		[Team.BLU]: newTeamTotals(Team.BLU),
	};
	for (const sm of summaries) {
		const t = totals[sm.team];
		if (!t) {
			continue;
		}
		t.players += 1;
		t.kills += sm.kills;
		t.assists += sm.assists;
		t.deaths += sm.deaths;
		t.damage += sm.damage;
		t.damageTaken += sm.dt;
		t.healing += sm.healing;
		t.charges += sm.charges;
		t.drops += sm.drops;
		t.captures += sm.cap;
		t.capturesBlocked += sm.capturesBlocked;
	}
	return [totals[Team.BLU], totals[Team.RED]];
};

const roundForTick = (rounds: MatchRound[], tick: number): number => {
	for (const r of rounds) {
		if (tick >= r.tickStart && tick <= r.tickEnd) {
			return r.round;
		}
	}
	return 0;
};

const assembleKillFeed = (kills: MatchKill[], rounds: MatchRound[]): KillFeedEntry[] => {
	return (kills ?? [])
		.map((k) => ({
			tick: Number(k.tick),
			round: roundForTick(rounds, Number(k.tick)),
			killerSteamId: String(k.killerSteamId ?? ""),
			victimSteamId: String(k.victimSteamId ?? ""),
			weapon: k.weapon,
		}))
		.toSorted((a, b) => a.tick - b.tick);
};

const assembleChatFeed = (chat: MatchChatLog[], rounds: MatchRound[]): ChatFeedEntry[] => {
	let startTick = Number.MAX_SAFE_INTEGER;
	for (const r of rounds) {
		if (r.tickStart < startTick) {
			startTick = r.tickStart;
		}
	}
	if (startTick === Number.MAX_SAFE_INTEGER) {
		startTick = 0;
	}
	return (chat ?? [])
		.map((c) => {
			const tick = Number(c.demoTick);
			return {
				tick,
				round: roundForTick(rounds, tick),
				steamId: String(c.steamId ?? ""),
				name: c.name,
				body: c.body,
				ticksSinceStart: Math.max(0, tick - startTick),
			};
		})
		.toSorted((a, b) => a.tick - b.tick);
};

const newVariant = (variant: RoundPlayerVariant) => ({
	kills: 0,
	assists: 0,
	deaths: 0,
	healing: 0,
	name: variant.variant,
	isWeapon: true,
	airshots: 0,
	backstabKills: 0,
	backstabs: 0,
	chargesKritz: 0,
	chargesQuickfix: 0,
	chargesUber: 0,
	chargesVacc: 0,
	damage: 0,
	damageTaken: 0,
	dominated: 0,
	dominations: 0,
	drops: 0,
	headshotKills: 0,
	headshots: 0,
	nearFullChargeDeath: 0,
	revenged: 0,
	revenges: 0,
});
