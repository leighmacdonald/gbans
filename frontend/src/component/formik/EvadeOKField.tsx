import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Tooltip from '@mui/material/Tooltip';
import { useFormikContext } from 'formik';

interface EvadeOKFieldValue {
    evade_ok: boolean;
}

export const EvadeOKField = () => {
    const { values, handleChange } = useFormikContext<EvadeOKFieldValue>();
    return (
        <FormGroup>
            <Tooltip
                title={
                    'Periodically update known friends lists and include them in the ban'
                }
            >
                <FormControlLabel
                    control={<Checkbox checked={values.evade_ok} />}
                    label="Evade OK"
                    name={'evade_ok'}
                    onChange={handleChange}
                />
            </Tooltip>
        </FormGroup>
    );
};
