import { z } from "zod/v4";
import { schemaTimeStamped } from "./chrono.ts";
import { PermissionLevelEnum } from "./people.ts";

export const schemaPage = z
	.object({
		slug: z.string(),
		body_md: z.string(),
		permission_level: PermissionLevelEnum,
		revision: z.number().optional(),
	})
	.merge(schemaTimeStamped);

export type Page = z.infer<typeof schemaPage>;
