import React from "react";
import {Redirect} from "react-router";

export const LoginSuccess = () => {
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get("token");
    if (token != null && token.length > 0) {
        localStorage.setItem("token", token);
        console.log(`Set token: ${token}`)
    }
    let next_url = urlParams.get("next_url");
    if (next_url == null || next_url == "") {
        next_url = "/"
    }
    return <Redirect to={next_url} />
}