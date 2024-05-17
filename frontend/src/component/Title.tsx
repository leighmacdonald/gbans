import { useEffect, useRef } from 'react';

interface TitleProps {
    // Must be a single string! See https://react.dev/reference/react-dom/components/title#use-variables-in-the-title
    children: string;
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
    const originalTitle = useRef<string | undefined>();
    useEffect(() => {
        if (originalTitle.current === undefined) {
            originalTitle.current = document.title;
        }

        document.title = `${children} | ${__SITE_NAME__}`;

        return () => {
            document.title = originalTitle.current!;
        };
    }, [originalTitle, children]);
    return null;
};
