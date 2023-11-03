import React, { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import { apiSaveFilter, Filter } from '../../api/filters';
import { Heading } from '../Heading';
import { FilterPatternField } from '../formik/FilterPatternField';
import { FilterTestField } from '../formik/FilterTestField';
import { IsRegexPatternField } from '../formik/IsRegexPatternField';
import { CancelButton, SaveButton } from './Buttons';

interface FilterEditModalProps {
    filter?: Filter;
}

interface FilterEditFormValues {
    filter_id?: number;
    pattern: RegExp | string;
    is_regex: boolean;
    is_enabled?: boolean;
}

export const FilterEditModal = NiceModal.create(
    ({ filter }: FilterEditModalProps) => {
        const modal = useModal();

        const onSave = useCallback(
            async (values: FilterEditFormValues) => {
                try {
                    const resp = await apiSaveFilter({
                        is_enabled: values.is_enabled,
                        filter_id: values.filter_id,
                        is_regex: values.is_regex,
                        pattern: values.pattern
                    });
                    modal.resolve(resp);
                    await modal.hide();
                    window.location.reload();
                } catch (e) {
                    modal.reject(e);
                }
            },
            [modal]
        );

        return (
            <Formik<Filter>
                onSubmit={onSave}
                initialValues={{
                    pattern: filter?.pattern ?? '',
                    is_regex: filter?.is_regex ?? false,
                    filter_id: filter?.filter_id ?? undefined,
                    author_id: filter?.author_id ?? undefined,
                    is_enabled: filter?.is_enabled ?? true
                }}
            >
                <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
                    <DialogTitle component={Heading}>Filter Editor</DialogTitle>
                    <DialogContent>
                        <Stack spacing={2}>
                            <FilterPatternField />
                            <IsRegexPatternField />
                            <FilterTestField />
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <SaveButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);
