import { createRef, useMemo } from 'react';
import {
    AdmonitionDirectiveDescriptor,
    BoldItalicUnderlineToggles,
    codeBlockPlugin,
    CodeToggle,
    directivesPlugin,
    frontmatterPlugin,
    imagePlugin,
    InsertImage,
    InsertTable,
    InsertThematicBreak,
    linkDialogPlugin,
    linkPlugin,
    listsPlugin,
    ListsToggle,
    markdownShortcutPlugin,
    MDXEditor,
    MDXEditorMethods,
    quotePlugin,
    tablePlugin,
    thematicBreakPlugin,
    toolbarPlugin,
    UndoRedo
} from '@mdxeditor/editor';
import { headingsPlugin } from '@mdxeditor/editor';
import '@mdxeditor/editor/style.css';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { apiSaveAsset, assetURL } from '../../api/media.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import './MarkdownField.css';
import { FieldProps } from './common.ts';

type MDBodyFieldProps = {
    fileUpload?: boolean;
    minHeight?: number;
    rows?: number;
} & FieldProps;

const imageUploadHandler = async (media: File) => {
    const resp = await apiSaveAsset(media);
    return assetURL(resp);
};

/**
 * Should be used to call the markdown editor methods. Mostly useful for clearing the current value after
 * a successful form submission.
 */
// eslint-disable-next-line react-refresh/only-export-components
export const mdEditorRef = createRef<MDXEditorMethods>();

/**
 * Uses MDXEditor for markdown formatting wysiwyg editing. https://mdxeditor.dev/editor/docs/getting-started
 *
 * To clear it after a successful submission: mdEditorRef.current?.setMarkdown('');
 *
 * @param state
 * @param handleChange
 * @constructor
 */
export const MarkdownField = ({ state, handleChange }: MDBodyFieldProps) => {
    const { sendFlash } = useUserFlashCtx();
    const theme = useTheme();

    const onError = (payload: { error: string; source: string }) => {
        sendFlash('error', payload.error);
    };

    const classes = useMemo(() => {
        if (theme.mode == 'dark') {
            return 'dark-theme md-editor dark-editor mdxeditor-root-contenteditable-dark';
        } else {
            return 'md-editor light-editor mdxeditor-root-contenteditable-light';
        }
    }, [theme.mode]);

    const errInfo = useMemo(() => {
        return state.meta.touchedErrors.length > 0 ? (
            <Typography padding={1} color={theme.palette.error.main}>
                {state.meta.touchedErrors}
            </Typography>
        ) : (
            <></>
        );
    }, [state.meta.touchedErrors, theme.palette.error.main]);

    return (
        <Paper>
            <MDXEditor
                contentEditableClassName={'md-content-editable'}
                className={classes}
                autoFocus={true}
                markdown={state.value.trimEnd()}
                placeholder={'Message (Min length: 10 characters)'}
                plugins={[
                    toolbarPlugin({
                        toolbarContents: () => (
                            <Stack direction={'row'}>
                                <UndoRedo />
                                <BoldItalicUnderlineToggles />
                                <CodeToggle />
                                <InsertImage />
                                <InsertTable />
                                <InsertThematicBreak />
                                <ListsToggle />
                            </Stack>
                        )
                    }),
                    listsPlugin(),
                    quotePlugin(),
                    headingsPlugin(),
                    linkPlugin(),
                    linkDialogPlugin(),
                    imagePlugin({ imageUploadHandler }),
                    tablePlugin(),
                    thematicBreakPlugin(),
                    frontmatterPlugin(),
                    codeBlockPlugin({ defaultCodeBlockLanguage: 'txt' }),
                    directivesPlugin({
                        directiveDescriptors: [AdmonitionDirectiveDescriptor]
                    }),
                    markdownShortcutPlugin()
                ]}
                onError={onError}
                onChange={handleChange}
                ref={mdEditorRef}
            />
            {errInfo}
        </Paper>
    );
};
