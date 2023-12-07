export interface hookResult<T> {
    data: T;
    count: number;
    loading: boolean;
    error?: string;
}
