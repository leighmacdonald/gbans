import { z } from 'zod';

export const schemaTimeStamped = z.object({
    created_on: z.date(),
    updated_on: z.date(),
    valid_until: z.date().optional()
});

export const schemaDateRange = z.object({
    date_start: z.date(),
    date_end: z.date()
});

export type DateRange = z.infer<typeof schemaDateRange>;
