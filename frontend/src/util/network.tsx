export interface apiResponse<T> {
    status: boolean
    resp: Response
    json: T | apiError
}

export interface apiError {
    error?: string
}

export const apiCall = async <TResponse, TRequestBody = any>(url: string, method: string, body?: TRequestBody): Promise<apiResponse<TResponse>> => {
    const headers: Record<string, string> = {
        "Content-Type": "application/json; charset=UTF-8"
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
    opts.headers = headers
    const resp = await fetch(url, opts)
    if (!resp.status) {
        throw apiErr("Invalid response code", resp)
    }
    const json = ((await resp.json() as TResponse) as any).data
    if (json?.error && json.error !== "") {
        throw apiErr(`Error received: ${json.error}`, resp)
    }
    return {json: json, resp: resp, status: resp.ok}
}

class ApiException extends Error {
    public resp: Response
    constructor(msg: string, response: Response) {
        super(msg);
        this.resp = response
    }
}

const apiErr = (msg: string, resp: Response): ApiException => {
    return new ApiException(msg, resp);
}