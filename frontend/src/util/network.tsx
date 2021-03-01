export interface apiResponse<T> {
    status: boolean
    resp: Response
    json: T | apiError
}

export interface apiError {
    message: string
}

export async function apiCall<TResponse, TRequestBody = any>(url: string, method: string, body?: TRequestBody): Promise<apiResponse<TResponse>> {
    const headers: Record<string, string> = {
        "Content-type": "application/json; charset=UTF-8"
    }
    let opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = localStorage.getItem("token");
    if (token != "") {
        headers["Authorization"] = `Bearer ${token}`
    }
    if (method === "POST" && body) {
        opts["body"] = JSON.stringify(body)
    }
    opts.headers = headers;
    const resp = await fetch(url, opts);
    const json = ((await resp.json() as TResponse) as any).data;
    return {json: json, resp: resp, status: resp.ok}
}