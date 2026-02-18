import { useEffect, useRef } from "react";
import { useAppInfoCtx } from "../contexts/AppInfoCtx.ts";

interface TitleProps {
	// Must be a single string! See https://react.dev/reference/react-dom/components/title#use-variables-in-the-title
	children?: string;
}

// Sets the window title to the passed in children.
//
// Examples:
//   - <Title>Really cool page</Title>
//   - <Title>{`Page ${page_number}`}</Title>
//
// TODO: Bad things may happen if a route tries to render multiple
// <Title>s!
export const Title = ({ children }: TitleProps) => {
	const { appInfo } = useAppInfoCtx();
	const originalTitle = useRef<string | undefined>("");

	useEffect(() => {
		if (originalTitle.current === undefined) {
			originalTitle.current = document.title;
		}

		if (children) {
			document.title = `${children} | ${appInfo.site_name}`;
		} else {
			document.title = `${appInfo.site_name}`;
		}

		return () => {
			document.title = originalTitle.current ?? document.title;
		};
	}, [children, appInfo.site_name]);
	return null;
};
