import { expect, test } from "vitest";
import { once } from "./once.ts";

test("concurrent calls with the same key share a single run", async () => {
	let calls = 0;

	const run = async () => {
		calls += 1;
	};

	const a = once("k", run);
	const b = once("k", run);

	expect(a).toBe(b);
	await a;

	expect(calls).toBe(1);
});

test("different keys run independently", async () => {
	let calls = 0;

	const run = async () => {
		calls += 1;
	};

	await Promise.all([once("a", run), once("b", run)]);

	expect(calls).toBe(2);
});

test("the key is released after success", async () => {
	let calls = 0;

	const run = async () => {
		calls += 1;
	};

	await once("k", run);
	await once("k", run);

	expect(calls).toBe(2);
});

test("the key is released after failure", async () => {
	let calls = 0;

	const failing = async () => {
		calls += 1;
		throw new Error("boom");
	};

	await expect(once("k", failing)).rejects.toThrow("boom");
	await expect(once("k", failing)).rejects.toThrow("boom");

	expect(calls).toBe(2);
});

test("a pending run is shared, a fresh run starts after it settles", async () => {
	let calls = 0;
	let release: () => void;

	const gate = new Promise<void>((resolve) => {
		release = resolve;
	});

	const first = once("k", async () => {
		calls += 1;
		await gate;
	});
	const second = once("k", async () => {
		calls += 1;
	});

	expect(second).toBe(first);

	release?.();
	await first;
	expect(calls).toBe(1);

	const third = once("k", async () => {
		calls += 1;
	});

	expect(third).not.toBe(first);
	await third;
	expect(calls).toBe(2);
});
