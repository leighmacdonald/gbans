import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import type { PersonDisplay } from "../../rpc/person/v1/person_core_pb";
import {
	type Match,
	type MatchChatLog,
	type RoundPlayer,
	type RoundPlayerVariant,
	Team,
} from "../../rpc/stats/v1/stats_pb";
import { classList } from "../../tf2";
import { emptyOrNullString } from "../../util/types";

export type MatchRow = {
	player: PersonDisplay;
	team: string;
	classes: string[];
	points: number;
	kills: number;
	assists: number;
	deaths: number;
	damage: number;
	kad: number;
	kd: number;
	dt: number;
	// dtm: number;
	hp: number;
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

export type MatchView = {
	info: MatchInfo;
	summaries: MatchRow[];
	rounds: MatchRound[];
	chat: MatchChatLog[];
	variants: Record<string, Record<string, MatchPlayerVariantStats>>;
};

export const assembleMatch = (data: Match): MatchView => {
	if (!data.overview) {
		throw "invalid overview";
	}
	const summaries: Record<string, MatchRow> = {};
	//const players = data.players;
	const rounds: MatchRound[] = [];

	for (let i = 0; i < data.rounds.length; i++) {
		if (Number(data.rounds[i].durationMs) === 0) {
			continue;
		}
		rounds.push({
			round: i + 1,
			winner: data.rounds[i].winner,
			durationMs: Number(data.rounds[i].durationMs),
			isStalemate: data.rounds[i].isStalemate,
			isSuddenDeath: data.rounds[i].isSuddenDeath,
			players: data.rounds[i].players,
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
			// summaries[data.rounds[i].players[p].steamId].name = data.rounds[i].players[p].;
			sm.team = Team[rp.team];
		}
	}

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
		summaries: Object.values(summaries).toSorted((a, b) => b.points - a.points),
		chat: data.chatLogs,
		variants: assemblePlayerVariants(data),
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
	deaths: 0,
	dt: 0,
	// dtm: 0,
	hp: 0,
	kad: 0,
	kd: 0,
	kills: 0,
	hs: 0,
	player: po,
	healing: 0,
	capturesBlocked: 0,
	team: Team[Team.UNASSIGNED_UNSPECIFIED],
});

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
