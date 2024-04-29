import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Tooltip from '@mui/material/Tooltip';
import { useFormikContext } from 'formik';

interface StatsHiddenFieldValue {
    stats_hidden: boolean;
}

export const StatsHiddenField = () => {
    const { values, handleChange } = useFormikContext<StatsHiddenFieldValue>();
    return (
        <FormGroup>
            <Tooltip title={'Enable your game stats to be shown to the public'}>
                <FormControlLabel
                    control={<Checkbox checked={values.stats_hidden} />}
                    label="Hide Game Stats"
                    name={'stats_hidden'}
                    onChange={handleChange}
                />
            </Tooltip>
        </FormGroup>
    );
};
