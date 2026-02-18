import type { ApiError } from "../error.tsx";
import { readAccessToken } from "../util/auth/readAccessToken.ts";
import { emptyOrNullString } from "../util/types";

export interface DataCount {
	count: number;
}

export class EmptyBody {}

export const apiRootURL = (): string => `${location.protocol}//${location.host}`;

type httpMethods = "POST" | "GET" | "DELETE" | "PUT";

/**
 * All api requests are handled through this interface.
 *
 * @param url
 * @param method
 * @param body
 * @param abortController
 * @param isFormData
 * @throws AppError
 */
export const apiCall = async <TResponse = EmptyBody | null, TRequestBody = Record<string, unknown> | object>(
	url: string,
	method: httpMethods = "GET",
	body?: TRequestBody | undefined | FormData | Record<string, string>,
	abortController?: AbortController,
	isFormData: boolean = false,
): Promise<TResponse> => {
	const headers: Record<string, string> = {};
	const requestOptions: RequestInit = {
		mode: "cors",
		credentials: "include",
		method: method.toUpperCase(),
	};

	const accessToken = readAccessToken();

	if (!emptyOrNullString(accessToken)) {
		headers.Authorization = `Bearer ${accessToken}`;
	}

	if (!isFormData) {
		headers["Content-Type"] = "application/json; charset=UTF-8";
	}

	requestOptions.headers = headers;

	if (method !== "GET" && body) {
		requestOptions.body = isFormData ? (body as FormData) : JSON.stringify(body);
	}

	if (abortController !== undefined) {
		requestOptions.signal = abortController.signal;
	}

	const fullURL = new URL(url, apiRootURL());
	if (method === "GET" && body) {
		fullURL.search = new URLSearchParams(body as Record<string, string>).toString();
	}

	const response = await fetch(fullURL, requestOptions);

	if (!response.ok) {
		throw (await response.json()) as ApiError;
	}

	if (response.status === 204) {
		return null as TResponse;
	}

	return (await response.json()) as TResponse;
};
