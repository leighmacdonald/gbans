
export type Nullable<T> = T | null | undefined

export type NotNull<T> = T extends (null | undefined) ? never : T