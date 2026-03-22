import type { Asset, UserUploadedFile } from "../schema/asset.ts";
import { transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";

export const assetURL = (asset: Asset): string => `/asset/${asset.asset_id}`;

export const apiSaveAsset = async (file: File, name = "", signal: AbortSignal) => {
	const imageData = new FormData();
	imageData.append("file", file);
	imageData.append("name", name);

	return transformTimeStampedDates(await apiCall<Asset>(signal, `/api/asset`, "POST", imageData, true));
};

export const apiSaveContestEntryMedia = async (contest_id: string, upload: UserUploadedFile, signal: AbortSignal) =>
	await apiCall<Asset, UserUploadedFile>(signal, `/api/contests/${contest_id}/upload`, "POST", upload);
