export type Nullable<T> = T | null | undefined;

export type NotNull<T> = T extends null | undefined ? never : T;

export const emptyOrNullString = (value: string | null | undefined) => {
    return value == null || value == '' || value == undefined;
};

export const StringIsNumber = (value: unknown) => !isNaN(Number(value));

export const EnumToArray = (obj: Record<string, string>) =>
    Object.keys(obj)
        .filter(StringIsNumber)
        .map((key) => obj[key]);

export interface IpRecord {
    IP: string;
}
