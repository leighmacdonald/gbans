import { apiCall, TimeStamped } from './common';

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

export interface Asset {
    asset_id: string;
    bucket: string;
    path: string;
    name: string;
    mime_type: string;
    size: number;
    old_id: number;
}

export interface MediaUploadResponse extends BaseUploadedMedia {
    url: string;
}

export interface UserUploadedFile {
    content: string;
    name: string;
    mime: string;
    size: number;
}

export const apiSaveMedia = async (upload: UserUploadedFile) =>
    await apiCall<MediaUploadResponse, UserUploadedFile>(
        `/api/media`,
        'POST',
        upload
    );
