import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";

/** One Hammer unit equals one inch, or 0.0254 metres. */
export const HAMMER_UNIT_IN_METERS = 0.0254;

/** Fraction of the overview's largest dimension used as viewport buffer. */
export const OVERVIEW_VIEWPORT_PADDING_RATIO = 0.08;

export type ImageSize = {
	width: number;
	height: number;
};

export type PixelBounds = {
	minX: number;
	minY: number;
	maxX: number;
	maxY: number;
};

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
 * Kill positions use Hammer units. The tf2-maps world files and their .prj
 * sidecars use metres, so convert before applying the world-file transform.
 * px grows right, py grows down from the top-left pixel.
 */
export const worldToPixel = (wf: WorldFile, x: number, y: number): { px: number; py: number } => {
	const xMetres = x * HAMMER_UNIT_IN_METERS;
	const yMetres = y * HAMMER_UNIT_IN_METERS;
	return {
		px: (xMetres - wf.topLeftX) / wf.pixelSizeX,
		py: (wf.topLeftY - yMetres) / Math.abs(wf.pixelSizeY),
	};
};

export const overviewPixelBounds = (size: ImageSize): PixelBounds => ({
	minX: 0,
	minY: 0,
	maxX: size.width,
	maxY: size.height,
});

export const paddedOverviewPixelBounds = (size: ImageSize): PixelBounds => {
	const padding = Math.max(size.width, size.height) * OVERVIEW_VIEWPORT_PADDING_RATIO;
	return {
		minX: -padding,
		minY: -padding,
		maxX: size.width + padding,
		maxY: size.height + padding,
	};
};

export const pixelBoundsToLatLng = (imageHeight: number, bounds: PixelBounds): [[number, number], [number, number]] => [
	[imageHeight - bounds.minY, bounds.minX],
	[imageHeight - bounds.maxY, bounds.maxX],
];

export type ReferencePoint = {
	classname: string;
	targetname: string;
	team: string;
	x: number;
	y: number;
	z: number;
};

export type ReferenceMarkerStyle = {
	radius: number;
	color: string;
	fillColor: string;
	fillOpacity: number;
	weight: number;
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
	typeof value === "object" && value !== null && !Array.isArray(value);

/**
 * Parse a tf2-maps reference FeatureCollection. Geometry coordinates are a
 * pseudo-WGS84 preview projection, so use the Hammer-unit hx/hy properties and
 * project them with worldToPixel.
 */
export const parseReferenceFile = (data: unknown): ReferencePoint[] | null => {
	if (!isRecord(data) || data.type !== "FeatureCollection" || !Array.isArray(data.features)) {
		return null;
	}
	const points: ReferencePoint[] = [];
	for (const feature of data.features) {
		if (!isRecord(feature) || !isRecord(feature.geometry) || feature.geometry.type !== "Point") {
			continue;
		}
		if (!isRecord(feature.properties)) {
			continue;
		}
		const x = Number(feature.properties.hx);
		const y = Number(feature.properties.hy);
		const z = Number(feature.properties.hz ?? 0);
		if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(z)) {
			continue;
		}
		points.push({
			classname: String(feature.properties.classname ?? "unknown"),
			targetname: String(feature.properties.targetname ?? ""),
			team: String(feature.properties.team ?? ""),
			x,
			y,
			z,
		});
	}
	return points;
};

export const REFERENCE_HEALTH_COLOR = "#00c853";
export const REFERENCE_AMMO_COLOR = "#ff9100";

export const referencePointLabel = (point: ReferencePoint): string => {
	const name = point.targetname || point.classname;
	return point.team === "" ? name : `${name} · team ${point.team}`;
};

export const referencePointStyle = (point: ReferencePoint): ReferenceMarkerStyle => {
	if (point.classname === "team_control_point") {
		return { radius: 5, color: "#ffffff", fillColor: "#ffffff", fillOpacity: 0.9, weight: 1.5 };
	}
	if (point.classname.startsWith("item_healthkit_")) {
		return {
			radius: 2.5,
			color: REFERENCE_HEALTH_COLOR,
			fillColor: REFERENCE_HEALTH_COLOR,
			fillOpacity: 0.85,
			weight: 1,
		};
	}
	if (point.classname.startsWith("item_ammopack_")) {
		return {
			radius: 2.5,
			color: REFERENCE_AMMO_COLOR,
			fillColor: REFERENCE_AMMO_COLOR,
			fillOpacity: 0.85,
			weight: 1,
		};
	}
	if (point.classname.startsWith("item_")) {
		return { radius: 2.5, color: "#ffd54f", fillColor: "#ffd54f", fillOpacity: 0.8, weight: 1 };
	}
	if (point.team === "3") {
		return { radius: 2.5, color: blu, fillColor: blu, fillOpacity: 0.75, weight: 1 };
	}
	if (point.team === "2") {
		return { radius: 2.5, color: red, fillColor: red, fillOpacity: 0.75, weight: 1 };
	}
	return { radius: 2.5, color: "#9e9e9e", fillColor: "#9e9e9e", fillOpacity: 0.7, weight: 1 };
};

export const overviewUrls = (mapName: string): { image: string; worldFile: string } => ({
	image: `/maps/${mapName}_overview.png`,
	worldFile: `/maps/${mapName}_overview.pgw`,
});

export const referenceUrls = (mapName: string): { primary: string; fallback: string } => ({
	primary: `/maps/${mapName}_reference.geojson`,
	fallback: `/maps/${mapName}_reference.json`,
});

export const fetchReferenceFile = async (urls: {
	primary: string;
	fallback: string;
}): Promise<ReferencePoint[] | null> => {
	for (const url of [urls.primary, urls.fallback]) {
		try {
			const res = await fetch(url);
			if (!res.ok) {
				continue;
			}
			const parsed = parseReferenceFile(await res.json());
			if (parsed) {
				return parsed;
			}
		} catch {
			// Try the fallback reference filename below.
		}
	}
	return null;
};

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
