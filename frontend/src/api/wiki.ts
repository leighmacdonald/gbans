import type { Page } from "../schema/wiki.ts";
import { apiCall } from "./common";

export const apiGetWikiPage = async (slug: string, signal: AbortSignal) =>
	await apiCall<Page>(signal, `/api/wiki/slug/${slug}`);

export const apiSaveWikiPage = async (page: Page, signal: AbortSignal) =>
	await apiCall<Page>(signal, `/api/wiki/slug/${page.slug}`, "PUT", page);
