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
import { FormHelperText } from '@mui/material';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { TextFieldProps } from '@mui/material/TextField';
import { useTheme } from '@mui/material/styles';
import * as Sentry from '@sentry/react';
import { useStore } from '@tanstack/react-form';
import { apiSaveAsset, assetURL } from '../../../api/media.ts';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../../hooks/useUserFlashCtx.ts';
import { logErr } from '../../../util/errors.ts';
import { errorDialog } from '../../ErrorBoundary.tsx';
import './MarkdownField.css';

export type MDBodyFieldProps = {
    fileUpload?: boolean;
    label: string;
} & TextFieldProps;

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
 * Uses MDXEditor for Markdown formatting wysiwyg editing. https://mdxeditor.dev/editor/docs/getting-started
 *
 * To clear it after a successful submission: mdEditorRef.current?.setMarkdown('');
 *
 */
export const MarkdownField = (props: MDBodyFieldProps) => {
    const field = useFieldContext<string>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    const { sendFlash } = useUserFlashCtx();
    const theme = useTheme();

    const onError = (payload: { error: string; source: string }) => {
        logErr(payload);
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
        return errors.length > 0 ? (
            <FormHelperText error={true}>{errors.map((e) => e.message).join(', ')}</FormHelperText>
        ) : (
            <FormHelperText>{props.helperText}</FormHelperText>
        );
    }, [errors, theme.palette.error.main]);

    return (
        <Paper>
            <Sentry.ErrorBoundary showDialog={true} fallback={errorDialog}>
                <MDXEditor
                    contentEditableClassName={'md-content-editable'}
                    className={classes}
                    autoFocus={true}
                    markdown={field.state.value}
                    placeholder={props.placeholder ?? 'Message (Min length: 10 characters)'}
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
                    // onChange={(c) => {
                    //     field.setValue(c);
                    // }}
                    // onBlur={(v) => {
                    //     alert('fix2');
                    //     field.setValue()
                    // }}
                    ref={mdEditorRef}
                />
                {errInfo}
            </Sentry.ErrorBoundary>
        </Paper>
    );
};
