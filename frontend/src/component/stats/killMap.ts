import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";

/** Parsed ESRI world file (.pgw) accompanying a TF2 overview image. */
export type WorldFile = {
	/** World units per pixel along x (positive). */
	pixelSizeX: number;
	/** World units per pixel along y (negative in TF2 overviews). */
	pixelSizeY: number;
	/** World x of the top-left pixel. */
	topLeftX: number;
	/** World y of the top-left pixel. */
	topLeftY: number;
};

/**
 * Parse the 6-line world file format. Returns null on malformed input.
 */
export const parseWorldFile = (text: string): WorldFile | null => {
	const lines = text
		.split("\n")
		.map((l) => l.trim())
		.filter((l) => l !== "");
	if (lines.length < 6) {
		return null;
	}
	const nums = lines.slice(0, 6).map(Number);
	if (nums.some((n) => !Number.isFinite(n))) {
		return null;
	}
	const [pixelSizeX, , , pixelSizeY, topLeftX, topLeftY] = nums;
	if (pixelSizeX === 0 || pixelSizeY === 0) {
		return null;
	}
	return { pixelSizeX, pixelSizeY, topLeftX, topLeftY };
};

/**
 * Convert TF2 world coordinates to overview image pixel coordinates.
 * px grows right, py grows down from the top-left pixel.
 */
export const worldToPixel = (wf: WorldFile, x: number, y: number): { px: number; py: number } => ({
	px: (x - wf.topLeftX) / wf.pixelSizeX,
	py: (wf.topLeftY - y) / Math.abs(wf.pixelSizeY),
});

export const overviewUrls = (mapName: string): { image: string; worldFile: string } => ({
	image: `/maps/${mapName}_overview.png`,
	worldFile: `/maps/${mapName}_overview.pgw`,
});

export const fetchWorldFile = async (url: string): Promise<WorldFile | null> => {
	try {
		const res = await fetch(url);
		if (!res.ok) {
			return null;
		}
		return parseWorldFile(await res.text());
	} catch {
		return null;
	}
};

/**
 * Deterministic, unique-ish line color per player derived from their steamId.
 */
export const playerColor = (steamId: string): string => {
	let hash = 0x811c9dc5;
	for (let i = 0; i < steamId.length; i++) {
		hash ^= steamId.charCodeAt(i);
		hash = Math.imul(hash, 0x01000193);
	}
	const hue = Math.abs(hash) % 360;
	return `hsl(${hue}, 75%, 50%)`;
};

export const teamColorOf = (team: Team): string => (team === Team.BLU ? blu : red);

/** Highlight colors for the selected player's kills/deaths; vivid against team colors and dark overviews. */
export const HIGHLIGHT_KILL = "#c8ff00";
export const HIGHLIGHT_DEATH = "#ff2e88";
export const DIMMED_OPACITY = 0.25;
/** Line color for kills with no killer (world/environment). */
export const WORLD_COLOR = "#9e9e9e";
