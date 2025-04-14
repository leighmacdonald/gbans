import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import { Updater } from '@tanstack/react-form';

type Props = {
    readonly label?: string;
    state: {
        value: boolean;
    };
    handleChange: (updater: Updater<boolean>) => void;
    handleBlur: () => void;
};

export const CheckboxSimple = ({ label, handleChange, state, handleBlur }: Props) => {
    if (!state) {
        // FIXME state seems to not be initialized on first load for some reason.
        return <></>;
    }

    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox onChange={(_, v) => handleChange(v)} onBlur={handleBlur} checked={state.value} />}
                label={label}
            />
        </FormGroup>
    );
};
