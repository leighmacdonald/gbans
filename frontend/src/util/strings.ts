import type { Group, Override, SMUser } from "../rpc/sourcemod/v1/sourcemod_pb";
import { z } from "zod/v4";

const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

export const randomStringAlphaNum = (length: number) => {
	let result = "";

	for (let i = 0; i < length; i++) {
		result += characters.charAt(Math.floor(Math.random() * characters.length));
	}

	return result;
};

export const cidrHostCount = (cidr: string): number => {
	if (!cidr.includes("/")) {
		return 0;
	}
	const mask = parseInt(cidr.split("/")[1], 10);
	return mask === 32 ? 1 : mask === 31 ? 2 : 2 ** (32 - mask) - 2;
};

export const EMPTY_UUID = "feb4bf16-7f55-4cb4-923c-4de69a093b79";

export const Flags = z.enum([
	"z",
	"a",
	"b",
	"c",
	"d",
	"e",
	"f",
	"g",
	"h",
	"i",
	"j",
	"k",
	"l",
	"m",
	"n",
	"o",
	"p",
	"q",
	"r",
	"s",
	"t",
]);

export const schemaFlags = z.object({
	a: z.boolean(),
	b: z.boolean(),
	c: z.boolean(),
	d: z.boolean(),
	e: z.boolean(),
	f: z.boolean(),
	g: z.boolean(),
	h: z.boolean(),
	i: z.boolean(),
	j: z.boolean(),
	k: z.boolean(),
	l: z.boolean(),
	m: z.boolean(),
	n: z.boolean(),
	o: z.boolean(),
	p: z.boolean(),
	q: z.boolean(),
	r: z.boolean(),
	s: z.boolean(),
	t: z.boolean(),
	z: z.boolean(),
});

export type Flags = z.infer<typeof Flags>;

export const hasSMFlag = (flag: Flags, entity?: Group | SMUser | Override) => {
	return entity?.flags.includes(flag) ?? false;
};
