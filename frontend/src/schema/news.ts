import { z } from 'zod/v4';
import { schemaTimeStamped } from './chrono.ts';

export const schemaNewsEntry = z
    .object({
        news_id: z.number(),
        title: z.string(),
        body_md: z.string(),
        is_published: z.boolean()
    })
    .merge(schemaTimeStamped);

export type NewsEntry = z.infer<typeof schemaNewsEntry>;
