/**
 * Result<T, E> — an explicit success/failure container. Favoured over throwing
 * from use cases so callers are forced to handle both branches.
 */
export type Result<T, E = Error> =
  | { readonly ok: true; readonly value: T }
  | { readonly ok: false; readonly error: E };

export const Result = {
  ok<T>(value: T): Result<T, never> {
    return { ok: true, value };
  },
  err<E>(error: E): Result<never, E> {
    return { ok: false, error };
  },
  isOk<T, E>(r: Result<T, E>): r is { ok: true; value: T } {
    return r.ok;
  },
  isErr<T, E>(r: Result<T, E>): r is { ok: false; error: E } {
    return !r.ok;
  },
  async fromPromise<T>(
    p: Promise<T>,
  ): Promise<Result<T, Error>> {
    try {
      const value = await p;
      return Result.ok(value);
    } catch (e) {
      return Result.err(e instanceof Error ? e : new Error(String(e)));
    }
  },
};
