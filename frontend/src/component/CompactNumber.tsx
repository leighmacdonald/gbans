import Tooltip from "@mui/material/Tooltip";
import { humanCount } from "../util/strings.ts";

const toFiniteNumber = (value: unknown): number | null => {
	if (typeof value === "number") {
		return Number.isFinite(value) ? value : null;
	}
	if (typeof value === "bigint") {
		return Number(value);
	}
	if (typeof value === "string" && value.trim() !== "") {
		const numeric = Number(value);
		return Number.isFinite(numeric) ? numeric : null;
	}
	return null;
};

/**
 * Renders a compact count (1.1k, 2m) and exposes the exact value on hover.
 */
export const CompactNumber = ({ value }: { value: unknown }) => {
	const numeric = toFiniteNumber(value);
	if (numeric === null) {
		return <>{value === null || value === undefined ? "" : String(value)}</>;
	}

	const compact = humanCount(numeric);
	if (Math.abs(numeric) < 1000) {
		return <>{compact}</>;
	}

	return (
		<Tooltip title={numeric.toLocaleString("en")}>
			<span>{compact}</span>
		</Tooltip>
	);
};
