import { FormikHandlers, FormikState } from 'formik/dist/types';

export interface BaseFormikInputProps<T> {
    id?: string;
    label?: string;
    initialValue?: string;
    fullWidth?: boolean;
    isReadOnly?: boolean;
    formik: FormikState<T> & FormikHandlers;
}
