import { createContext } from "react";
import type { AuthContextProps } from "../auth.tsx";

export const AuthContext = createContext<AuthContextProps | null>(null);
