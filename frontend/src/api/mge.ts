import { MGEStat, QueryMGE } from "../schema/mge";
import { LazyResult } from "../util/table";
import { parseDateTime } from "../util/time";
import { apiCall } from "./common";

export const apiMGEOverall = async (signal: AbortSignal, opts: QueryMGE, ) => {
  const response = await apiCall<LazyResult<MGEStat>>(signal, '/api/mge/ratings/overall', 'GET', opts);
  response.data = response.data.map(s => ({ ...s, lastplayed: parseDateTime(s.lastplayed as unknown as string) }));
  return response;
};
