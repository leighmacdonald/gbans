import { intervalToDuration } from 'date-fns';
import { z } from 'zod';
import { DataCount } from '../api';

export enum RowsPerPage {
    Ten = 10,
    TwentyFive = 25,
    Fifty = 50,
    Hundred = 100
}

export const isPermanentBan = (start: Date, end: Date): boolean => {
    const dur = intervalToDuration({
        start,
        end
    });
    const { years } = dur;
    return years != null && years > 5;
};

export const commonTableSearchSchema = {
    page: z.number().optional().catch(0),
    rows: z.number().optional().catch(RowsPerPage.TwentyFive),
    sortOrder: z.enum(['desc', 'asc']).optional()
};

export type Order = 'asc' | 'desc';

export interface LazyResult<T> extends DataCount {
    data: T[];
}
