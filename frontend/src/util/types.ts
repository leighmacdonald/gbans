export type Nullable<T> = T | null | undefined;

export const emptyOrNullString = (value: string | null | undefined) => {
	return value == null || value === "" || value === undefined;
};

export const zeroStringUndefined = (value?: string): string | undefined => {
	return !emptyOrNullString(value) ? value : undefined;
};
