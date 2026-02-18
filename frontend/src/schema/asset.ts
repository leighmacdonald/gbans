import { z } from "zod/v4";

export const Action = z.enum(["ban", "kick", "gag"]);

export const ActionColl = [Action.enum.ban, Action.enum.kick, Action.enum.gag];

export const schemaAsset = z.object({
	asset_id: z.string(),
	bucket: z.string(),
	mime_type: z.string(),
	size: z.number(),
	name: z.string(),
	author_id: z.string(),
	is_private: z.boolean(),
	created_on: z.date(),
	updated_on: z.date(),
});

export type Asset = z.infer<typeof schemaAsset>;

export const MediaTypes = {
	video: 0,
	image: 1,
	other: 2,
} as const;

export const MediaTypesEnum = z.enum(MediaTypes);
export type MediaTypesEnum = z.infer<typeof MediaTypesEnum>;

export const mediaType = (mime_type: string): MediaTypesEnum => {
	if (mime_type.startsWith("image/")) {
		return MediaTypes.image;
	} else if (mime_type.startsWith("video/")) {
		return MediaTypes.video;
	} else {
		return MediaTypes.other;
	}
};

const uInt8Schema: z.ZodType<Uint8Array> = z.custom<Uint8Array>((val) => {
	return val instanceof Uint8Array;
});

export const schemaBaseUploadedMedia = z.object({
	media_id: z.number(),
	author_id: z.number(),
	mime_type: z.string(),
	size: z.number(),
	name: z.string(),
	contents: uInt8Schema,
	deleted: z.boolean(),
	asset: schemaAsset,
});

export type BaseUploadedMedia = z.infer<typeof schemaBaseUploadedMedia>;

export const schemaUserUploadedFile = z.object({
	content: z.string(),
	name: z.string(),
});

export type UserUploadedFile = z.infer<typeof schemaUserUploadedFile>;
