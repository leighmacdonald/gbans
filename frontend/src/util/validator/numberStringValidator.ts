import { z } from 'zod';

export const numberStringValidator = (min: number, max?: number) => {
    return (arg: string, ctx: z.RefinementCtx) => {
        const parsed = Number(arg);
        if (isNaN(parsed)) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: 'Not a number'
            });
            return false;
        }

        let validator = z.number().min(min);
        if (max != undefined) {
            validator = validator.max(max);
        }

        const result = validator.safeParse(parsed);
        if (result.error) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: result.error.errors.map((e) => e.message).join(', ')
            });
            return false;
        }

        return true;
    };
};
