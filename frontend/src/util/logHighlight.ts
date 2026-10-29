// Tokenizes raw srcds log lines into spans so the terminal can syntax
// highlight names, quoted values, numbers and timestamps.
export type LogTokenKind = "default" | "timestamp" | "player" | "quoted" | "steamid" | "address" | "number" | "keyword";

export interface LogToken {
	text: string;
	kind: LogTokenKind;
	// Present on player references that carry a parseable SteamID, e.g. "[U:1:68745073]".
	steamId?: string;
}

const numericQuoted = /^\d+(?:\.\d+)?$/;
const addressQuoted = /^(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?$/;
const playerSteamId = /\[U:1:\d+\]/;

const TOKEN_REGEX =
	/(L \d{2}\/\d{2}\/\d{4} - \d{2}:\d{2}:\d{2}: )|("[^"]*<[^"]*>")|(\[U:1:\d+\])|((?:\d{1,3}\.){3}\d{1,3}(?::\d+)?)|("(?:[^"\\]|\\.)*")|(\b\d+(?:\.\d+)?\b)|(\b(?:entered the game|changed role to|committed suicide|joined team|map change|changing map|disconnected|connected|succeeded|failed|killed|triggered|suicide|started|stopped|says|say|details)\b)/gi;

export function tokenizeLogLine(line: string): LogToken[] {
	const tokens: LogToken[] = [];
	let lastIndex = 0;

	TOKEN_REGEX.lastIndex = 0;
	let match: RegExpExecArray | null = TOKEN_REGEX.exec(line);
	while (match !== null) {
		const [full, timestamp, player, steamId, address, quoted, number, keyword] = match;
		if (match.index > lastIndex) {
			tokens.push({ text: line.slice(lastIndex, match.index), kind: "default" });
		}

		let kind: LogTokenKind = "default";
		if (timestamp !== undefined) {
			kind = "timestamp";
		} else if (player !== undefined) {
			kind = "player";
		} else if (steamId !== undefined) {
			kind = "steamid";
		} else if (address !== undefined) {
			kind = "address";
		} else if (quoted !== undefined) {
			// Quoted numeric values (damage, healing, round numbers) read better
			// as numbers, and quoted IP:port as addresses, than as plain text.
			const content = full.slice(1, -1);
			if (numericQuoted.test(content)) {
				kind = "number";
			} else if (addressQuoted.test(content)) {
				kind = "address";
			} else {
				kind = "quoted";
			}
		} else if (number !== undefined) {
			kind = "number";
		} else if (keyword !== undefined) {
			kind = "keyword";
		}

		const token: LogToken = { text: full, kind };
		if (kind === "player") {
			const steamId = full.match(playerSteamId)?.[0];
			if (steamId) {
				token.steamId = steamId;
			}
		}
		tokens.push(token);
		lastIndex = match.index + full.length;
		match = TOKEN_REGEX.exec(line);
	}

	if (lastIndex < line.length) {
		tokens.push({ text: line.slice(lastIndex), kind: "default" });
	}

	return tokens;
}
