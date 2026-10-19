import { create } from "@bufbuild/protobuf";
import type { MetaDescriptor } from "@tanstack/react-router";
import { describe, expect, test } from "vitest";
import { InfoResponseSchema } from "../rpc/config/v1/config_pb.ts";
import { pageMeta } from "./meta.ts";

const href = "https://gbans.example.com/servers";

const appInfo = (overrides: Partial<Record<string, string>> = {}) =>
	create(InfoResponseSchema, {
		siteName: "Test Site",
		siteDescription: "Test description",
		bannerImage: "",
		...overrides,
	});

const metaMap = (tags: MetaDescriptor[]): Map<string, string> => {
	const map = new Map<string, string>();
	for (const tag of tags) {
		const t = tag as Record<string, string>;
		const key = t.property ?? t.name ?? "title";
		map.set(key, t.content ?? t.title ?? "");
	}
	return map;
};

const countKey = (tags: MetaDescriptor[], key: string): number =>
	tags.filter((tag) => {
		const t = tag as Record<string, string>;
		return t.property === key || t.name === key;
	}).length;

describe("pageMeta", () => {
	test("falls back to site wide values", () => {
		const meta = metaMap(pageMeta(appInfo(), {}, href));
		expect(meta.get("title")).toBe("Test Site");
		expect(meta.get("og:title")).toBe("Test Site");
		expect(meta.get("og:site_name")).toBe("Test Site");
		expect(meta.get("og:type")).toBe("website");
		expect(meta.get("og:description")).toBe("Test description");
		expect(meta.get("description")).toBe("Test description");
		expect(meta.get("og:url")).toBe(href);
	});

	test("formats the page title with the site name", () => {
		const meta = metaMap(pageMeta(appInfo(), { title: "server browser" }, href));
		expect(meta.get("title")).toBe("Server browser - Test Site");
		expect(meta.get("og:title")).toBe("Server browser - Test Site");
		expect(meta.get("twitter:title")).toBe("Server browser - Test Site");
	});

	test("uses the page description and type when provided", () => {
		const meta = metaMap(pageMeta(appInfo(), { description: "A page", type: "profile" }, href));
		expect(meta.get("og:description")).toBe("A page");
		expect(meta.get("description")).toBe("A page");
		expect(meta.get("og:type")).toBe("profile");
	});

	test("resolves the default banner to an absolute URL", () => {
		const meta = metaMap(pageMeta(appInfo(), {}, href));
		expect(meta.get("og:image")).toBe("https://gbans.example.com/banner-default.webp");
		expect(meta.get("twitter:image")).toBe("https://gbans.example.com/banner-default.webp");
	});

	test("resolves configured banner assets to absolute URLs", () => {
		const meta = metaMap(pageMeta(appInfo({ bannerImage: "/asset/123" }), {}, href));
		expect(meta.get("og:image")).toBe("https://gbans.example.com/asset/123");
	});

	test("keeps absolute banner URLs", () => {
		const meta = metaMap(pageMeta(appInfo({ bannerImage: "https://cdn.example.com/banner.webp" }), {}, href));
		expect(meta.get("og:image")).toBe("https://cdn.example.com/banner.webp");
	});

	test("prefers an explicit image", () => {
		const meta = metaMap(pageMeta(appInfo(), { image: "/asset/remote/42" }, href));
		expect(meta.get("og:image")).toBe("https://gbans.example.com/asset/remote/42");
	});

	test("emits each og tag exactly once using the property attribute", () => {
		const tags = pageMeta(appInfo(), { title: "servers" }, href);
		expect(countKey(tags, "og:title")).toBe(1);
		expect(tags.some((tag) => (tag as Record<string, string>).name === "og:title")).toBe(false);
	});

	test("includes twitter card metadata", () => {
		const meta = metaMap(pageMeta(appInfo(), {}, href));
		expect(meta.get("twitter:card")).toBe("summary_large_image");
		expect(meta.get("twitter:description")).toBe("Test description");
	});
});
