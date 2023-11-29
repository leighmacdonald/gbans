import React from 'react';
import { MuiMarkdown } from 'mui-markdown';
import { Highlight, themes } from 'prism-react-renderer';

export const MarkDownRenderer = ({ body_md }: { body_md: string }) => {
    return (
        <MuiMarkdown
            Highlight={Highlight}
            themes={themes}
            prismTheme={themes.github}
        >
            {body_md}
        </MuiMarkdown>
    );
};
