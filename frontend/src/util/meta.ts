import type { MetaDescriptor } from "@tanstack/react-router";
import type { InfoResponse } from "../rpc/config/v1/config_pb.ts";
import { toTitleCase } from "./strings.ts";

const defaultSiteName = "gbans";
const defaultDescription = "gbans is a web application for managing Team Fortress 2 communities";
const defaultBanner = "/banner-default.webp";

export type PageMetaOptions = {
	type?: "website" | "profile" | "article";
	title?: string;
	description?: string;
	image?: string;
};

const absoluteURL = (url: string, base: string): string => {
	try {
		return new URL(url, base).href;
	} catch {
		return url;
	}
};

export function pageMeta(appInfo: InfoResponse, opts: PageMetaOptions = {}, href?: string): MetaDescriptor[] {
	const url = href ?? (typeof window === "undefined" ? "" : window.location.href);
	const siteName = appInfo.siteName || defaultSiteName;
	const title = opts.title ? `${toTitleCase(opts.title)} - ${siteName}` : siteName;
	const description = opts.description || appInfo.siteDescription || defaultDescription;
	const image = absoluteURL(opts.image || appInfo.bannerImage || defaultBanner, url);

	return [
		{ title },
		{ name: "description", content: description },
		{ property: "og:type", content: opts.type ?? "website" },
		{ property: "og:site_name", content: siteName },
		{ property: "og:title", content: title },
		{ property: "og:description", content: description },
		{ property: "og:image", content: image },
		{ property: "og:image:alt", content: siteName },
		{ property: "og:url", content: url },
		{ name: "twitter:card", content: "summary_large_image" },
		{ name: "twitter:title", content: title },
		{ name: "twitter:description", content: description },
		{ name: "twitter:image", content: image },
	];
}
