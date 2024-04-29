import { useCallback } from 'react';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import * as yup from 'yup';
import { apiCreateThreadReply, ForumMessage } from '../api/forum';
import { logErr } from '../util/errors';
import { MDBodyField } from './MDBodyField';
import { SubmitButton } from './modal/Buttons';

interface ThreadReplyValues {
    body_md: string;
}

const validationSchema = yup.object({
    body_md: yup.string().min(3, 'Message Too Short').required('Message is required')
});

export const ForumThreadReplyBox = ({
    forum_thread_id,
    onSuccess
}: {
    forum_thread_id: number;
    onSuccess: (message: ForumMessage) => void;
}) => {
    const onSubmit = useCallback(
        async (values: ThreadReplyValues, formikHelpers: FormikHelpers<ThreadReplyValues>) => {
            try {
                const message = await apiCreateThreadReply(forum_thread_id, values.body_md);
                onSuccess(message);
                formikHelpers.resetForm();
            } catch (e) {
                logErr(e);
            }
        },
        [forum_thread_id, onSuccess]
    );

    return (
        <Paper>
            <Formik<ThreadReplyValues>
                initialValues={{ body_md: '' }}
                onSubmit={onSubmit}
                validateOnChange={true}
                validateOnBlur={true}
                validationSchema={validationSchema}
            >
                <Stack spacing={1} padding={1}>
                    <MDBodyField />
                    <Stack direction={'row'} padding={1}>
                        <SubmitButton />
                    </Stack>
                </Stack>
            </Formik>
        </Paper>
    );
};
