import { z } from "zod/v4";
import { PermissionLevelEnum } from "./people.ts";

export const schemaPage = z.object({
	slug: z.string(),
	body_md: z.string(),
	permission_level: PermissionLevelEnum,
	revision: z.number().optional(),
	created_on: z.date(),
	updated_on: z.date(),
});

export type Page = z.infer<typeof schemaPage>;
