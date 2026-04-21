export interface Page<T> {
  readonly items: ReadonlyArray<T>;
  readonly page: number;
  readonly pageSize: number;
  readonly total: number;
  readonly hasMore: boolean;
}

export interface PaginationParams {
  readonly page?: number;
  readonly pageSize?: number;
}

export function emptyPage<T>(): Page<T> {
  return {
    items: [],
    page: 1,
    pageSize: 0,
    total: 0,
    hasMore: false,
  };
}
