import { expect, test } from "vitest";
import { tokenizeLogLine } from "./logHighlight.ts";

test("plain text yields a single default token", () => {
	expect(tokenizeLogLine("hello world")).toEqual([{ text: "hello world", kind: "default" }]);
});

test("empty line yields no tokens", () => {
	expect(tokenizeLogLine("")).toEqual([]);
});

test("timestamp prefix is dimmed", () => {
	const tokens = tokenizeLogLine('L 02/21/2021 - 06:22:23: "a<1><[U:1:1]><Red>" say "hi"');
	expect(tokens[0]).toEqual({ text: "L 02/21/2021 - 06:22:23: ", kind: "timestamp" });
});

test("quoted player references are highlighted, inner ids are not double tokenized", () => {
	const tokens = tokenizeLogLine(
		'L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "brass_beast"',
	);
	const kinds = tokens.map((t) => t.kind);
	expect(kinds).toEqual(["timestamp", "player", "default", "keyword", "default", "player", "default", "quoted"]);
	expect(tokens[1].text).toBe('"Hacksaw<12><[U:1:68745073]><Red>"');
	expect(tokens[5].text).toBe('"Dzefersons14<8><[U:1:1080653073]><Blue>"');
});

test("chat messages are quoted, say is a keyword", () => {
	const tokens = tokenizeLogLine('L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"');
	const kinds = tokens.map((t) => t.kind);
	expect(kinds).toEqual(["timestamp", "player", "default", "keyword", "default", "quoted"]);
});

test("quoted numeric values are numbers, other quoted text is quoted", () => {
	const tokens = tokenizeLogLine(
		'L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")',
	);
	const byText = new Map(tokens.map((t) => [t.text, t.kind]));
	expect(byText.get('"11"')).toBe("number");
	expect(byText.get('"damage"')).toBe("quoted");
	expect(byText.get('"syringegun_medic"')).toBe("quoted");
});

test("bare steam ids and ip addresses are dimmed", () => {
	const tokens = tokenizeLogLine(
		'L 02/21/2021 - 06:22:23: Kick Vote details:  VoteInitiatorSteamID: [U:1:0000001]  Valid: 1  Proxy: 0"',
	);
	const byText = new Map(tokens.map((t) => [t.text, t.kind]));
	expect(byText.get("[U:1:0000001]")).toBe("steamid");
	expect(byText.get("1")).toBe("number");
	expect(byText.get("details")).toBe("keyword");

	const connect = tokenizeLogLine(
		'L 05/16/2021 - 05:47:24: "amogus gaming<13><[U:1:1089803558]><>" connected, address "139.47.95.130:47949"',
	);
	const kinds = connect.map((t) => t.kind);
	expect(kinds).toContain("address");
});

test("multi-word keywords match", () => {
	const tokens = tokenizeLogLine('L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><>" entered the game');
	expect(tokens.map((t) => t.kind)).toContain("keyword");
	expect(tokens.find((t) => t.kind === "keyword")?.text).toBe("entered the game");
});

test("player tokens carry the raw steam id, anonymous references do not", () => {
	const tokens = tokenizeLogLine(
		'L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" connected, address "139.47.95.130:47949"',
	);
	const player = tokens.find((t) => t.kind === "player");
	expect(player?.text).toBe('"Hacksaw<12><[U:1:68745073]><Red>"');
	expect(player?.steamId).toBe("[U:1:68745073]");

	const anon = tokenizeLogLine('L 02/21/2021 - 06:22:23: "Anonymous Player<50><[A:1:1234567890]><>" connected');
	const anonPlayer = anon.find((t) => t.kind === "player");
	expect(anonPlayer?.steamId).toBeUndefined();
});

test("tokens reassemble to the original line", () => {
	const line =
		'L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab")';
	expect(
		tokenizeLogLine(line)
			.map((t) => t.text)
			.join(""),
	).toBe(line);
});
