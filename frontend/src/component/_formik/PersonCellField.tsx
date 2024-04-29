import { useCallback } from 'react';
import { useFormikContext } from 'formik';
import { PersonCell, PersonCellProps } from '../PersonCell';

export const PersonCellField = <T,>(props: PersonCellProps) => {
    const { setFieldValue, submitForm } = useFormikContext<T & PersonCellProps>();

    const onClick = useCallback(async () => {
        await setFieldValue('steam_id', props.steam_id);
        await submitForm();
    }, [props.steam_id, setFieldValue, submitForm]);

    return <PersonCell {...props} onClick={onClick} />;
};

export const PersonCellFieldNonInteractive = (props: PersonCellProps) => {
    return <PersonCell {...props} />;
};
