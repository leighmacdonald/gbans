import { apiCall, TimeStamped, transformTimeStampedDates } from './common';

export const assetURL = (asset: Asset): string => `/asset/${asset.asset_id}`;

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

export interface UserUploadedFile {
    content: string;
    name: string;
}

export const apiSaveAsset = async (file: File, name = '') => {
    const imageData = new FormData();
    imageData.append('file', file);
    imageData.append('name', name);

    return transformTimeStampedDates(await apiCall<Asset>(`/api/asset`, 'POST', imageData, undefined, true));
};

export const apiSaveContestEntryMedia = async (contest_id: number, upload: UserUploadedFile) =>
    await apiCall<Asset, UserUploadedFile>(`/api/contests/${contest_id}/upload`, 'POST', upload);
