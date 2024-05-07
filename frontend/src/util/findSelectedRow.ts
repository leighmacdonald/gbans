import { findSelectedRows } from './findSelectedRows.ts';

export const findSelectedRow = <T>(selection: object, array: T[], property: keyof T) => {
    try {
        const found = findSelectedRows<T>(selection, array, property);
        if (found) {
            return found[0];
        }
        return undefined;
    } catch {
        return undefined;
    }
};
