type json_handler = (json: string) => void;
export type error_handler = (error: string) => void;

export function http(url: string, method: string, body: any,
                     on_success: json_handler, on_error: error_handler | undefined) {
    let opts = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    }
    if (method === "POST") {
        opts["headers"] = {
            "Content-type": "application/json; charset=UTF-8"
        }
        if (body) {
            opts["body"] = JSON.stringify(body)
        }
    }
    fetch(url, opts as any)
        .then(response => {
            response.json().then(on_success)
        })
        .catch(on_error ? on_error : error => {
            console.log("Unhandled error: " + error)
        })
}