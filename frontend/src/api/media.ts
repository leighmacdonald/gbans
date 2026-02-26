import type { Asset, UserUploadedFile } from "../schema/asset.ts";
import { transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";

export const assetURL = (asset: Asset): string => `/asset/${asset.asset_id}`;

export const apiSaveAsset = async (file: File, name = "") => {
	const imageData = new FormData();
	imageData.append("file", file);
	imageData.append("name", name);

	return transformTimeStampedDates(await apiCall<Asset>(`/api/asset`, "POST", imageData, undefined, true));
};

export const apiSaveContestEntryMedia = async (contest_id: string, upload: UserUploadedFile) =>
	await apiCall<Asset, UserUploadedFile>(`/api/contests/${contest_id}/upload`, "POST", upload);
