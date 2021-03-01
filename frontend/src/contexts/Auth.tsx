import React, {createContext, useReducer} from "react";
import {PlayerSummary} from "../component/PlayerBanForm";
import {concat} from "lodash-es";

export interface AuthContext {
    token: string
    profile: PlayerSummary
    errors: string[]
}

export interface StoreProps {
    children: JSX.Element[]
}

export const AuthStore = ({children}: StoreProps) => {
    const [state] = useReducer<React.Reducer<AuthContext, Action>>(
        Reducer, initialState)
    return (
        <Auth.Provider value={state} children={children} />
    )
}

type ReducerActionType = "SET_TOKEN" | "SET_PROFILE" | "SET_ERROR"

export interface Action {
    type: ReducerActionType
    payload: any
}

export const Reducer = (state: AuthContext, action: Action): AuthContext => {
    switch (action.type) {
        case 'SET_TOKEN':
            return {
                ...state,
                token: action.payload
            };
        case 'SET_PROFILE':
            return {
                ...state,
                profile: action.payload
            };
        case 'SET_ERROR':
            return {
                ...state,
                errors: concat<string>(state.errors, action.payload)
            };
        default:
            return state;
    }
};

const initialState: AuthContext = {
    token: "",
    profile: {
        steam_id: 0,
        personaname: "Guest",
    },
    errors: []
}

export const Auth = createContext<AuthContext>(initialState)
