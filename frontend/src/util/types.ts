export type Nullable<T> = T | null | undefined;

export type NotNull<T> = T extends null | undefined ? never : T;

export const emptyOrNullString = (value: string | null | undefined) => {
    return value == null || value == '' || value == undefined;
};
