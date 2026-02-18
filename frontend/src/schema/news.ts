import { z } from "zod/v4";

export const schemaNewsEntry = z.object({
	news_id: z.number(),
	title: z.string(),
	body_md: z.string(),
	is_published: z.boolean(),
	created_on: z.date(),
	updated_on: z.date(),
});

export type NewsEntry = z.infer<typeof schemaNewsEntry>;
