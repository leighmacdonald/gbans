import { apiCall, TimeStamped } from './common';

export interface BaseUploadedMedia extends TimeStamped {
    author_id: number;
    mime_type: string;
    size: number;
    name: string;
    contents: Uint8Array;
    deleted: boolean;
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
