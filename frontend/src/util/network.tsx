export interface apiResponse<T> {
    status: boolean
    resp: Response
    json: T | apiError
}

export interface apiError {
    message: string
}

export async function http<T>(url: string, method: string, body?: any): Promise<apiResponse<T>> {
    let opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    if (method === "POST") {
        opts["headers"] = {
            "Content-type": "application/json; charset=UTF-8"
        }
        if (body) {
            opts["body"] = JSON.stringify(body)
        }
    }
    const resp = await fetch(url, opts);
    const json = await resp.json() as T;
    return {json: json, resp: resp, status: resp.ok}
}