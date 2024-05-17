import { apiCall, TimeStamped, transformTimeStampedDates } from './common';

const assetUrl = (bucket: string, asset: Asset): string => `${__ASSET_URL__}/${bucket}/${asset.name}`;

export const assetURLMedia = (asset: Asset) => assetUrl('media', asset);

export const assetURLDemo = (asset: Asset) => assetUrl('demo', asset);

export enum MediaTypes {
    video,
    image,
    other
}

export const mediaType = (mime_type: string): MediaTypes => {
    if (mime_type.startsWith('image/')) {
        return MediaTypes.image;
    } else if (mime_type.startsWith('video/')) {
        return MediaTypes.video;
    } else {
        return MediaTypes.other;
    }
};

export interface BaseUploadedMedia extends TimeStamped {
    media_id: number;
    author_id: number;
    mime_type: string;
    size: number;
    name: string;
    contents: Uint8Array;
    deleted: boolean;
    asset: Asset;
}

export type Asset = {
    asset_id: string;
    bucket: string;
    mime_type: string;
    size: number;
    name: string;
    author_id: string;
    is_private: boolean;
    updated_on: Date;
    created_on: Date;
};

export interface MediaUploadResponse extends BaseUploadedMedia {
    url: string;
}

export interface UserUploadedFile {
    content: string;
    name: string;
    mime: string;
    size: number;
}

export const apiSaveAsset = async (upload: UserUploadedFile) =>
    transformTimeStampedDates(await apiCall<MediaUploadResponse, UserUploadedFile>(`/api/asset`, 'POST', upload));

export const apiSaveContestEntryMedia = async (contest_id: number, upload: UserUploadedFile) =>
    await apiCall<MediaUploadResponse, UserUploadedFile>(`/api/contests/${contest_id}/upload`, 'POST', upload);
