import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import { apiSaveFilter, Filter, FilterAction } from '../../api/filters';
import { Heading } from '../Heading';
import { DurationStringField } from '../formik/DurationStringField';
import { FilterActionField } from '../formik/FilterActionField';
import { FilterPatternField } from '../formik/FilterPatternField';
import { FilterTestField } from '../formik/FilterTestField';
import { IsRegexPatternField } from '../formik/IsRegexPatternField';
import { WeightField } from '../formik/WeightField';
import { CancelButton, SubmitButton } from './Buttons';

interface FilterEditModalProps {
    filter?: Filter;
}

interface FilterEditFormValues {
    filter_id?: number;
    pattern: RegExp | string;
    is_regex: boolean;
    is_enabled?: boolean;
    action: FilterAction;
    duration: string;
    weight: number;
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
                        pattern: values.pattern,
                        action: values.action,
                        duration: values.duration,
                        weight: values.weight
                    });
                    modal.resolve(resp);
                    await modal.hide();
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
                    is_enabled: filter?.is_enabled ?? true,
                    duration: filter?.duration ?? '1w',
                    action: filter?.action ?? FilterAction.Mute,
                    weight: filter?.weight ?? 1
                }}
            >
                <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={<FilterAltIcon />}
                    >
                        Filter Editor
                    </DialogTitle>
                    <DialogContent>
                        <Stack spacing={2}>
                            <FilterPatternField />
                            <FilterActionField />
                            <DurationStringField />
                            <WeightField />
                            <IsRegexPatternField />
                            <FilterTestField />
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

export default FilterEditModal;
