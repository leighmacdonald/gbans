import { z } from 'zod';

export const schemaTimeStamped = z.object({
    created_on: z.date(),
    updated_on: z.date()
});

export const schemaTimeStampedWithValidUntil = z
    .object({
        valid_until: z.date()
    })
    .merge(schemaTimeStamped);

export const schemaDateRange = z.object({
    date_start: z.date(),
    date_end: z.date()
});

export type DateRange = z.infer<typeof schemaDateRange>;
