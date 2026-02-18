import { useContext } from "react";
import { UserFlashCtx } from "../contexts/UserFlashCtx.tsx";

export const useUserFlashCtx = () => useContext(UserFlashCtx);
