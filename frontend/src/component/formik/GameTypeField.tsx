import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export enum GameType {
    sixes = 'sixes',
    highlander = 'highlander',
    ultiduo = 'ultiduo'
}

export const GameTypes = [
    GameType.sixes,
    GameType.highlander,
    GameType.ultiduo
];

export const gameTypeValidator = yup
    .string()
    .test('checkGameType', 'Invalid game type selection', (gameType) => {
        return GameTypes.includes(gameType as GameType);
    })
    .label('Select a game type to play')
    .required('game type is required');

export const GameTypeField = ({
    formik
}: {
    formik: FormikState<{
        game_type: GameType;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="gameType-label">Game Type</InputLabel>
            <Select<GameType>
                fullWidth
                disabled={formik.isSubmitting}
                label={'Game Type'}
                labelId="game_type-label"
                id="game_type"
                name={'game_type'}
                value={formik.values.game_type}
                onChange={formik.handleChange}
                error={
                    formik.touched.game_type && Boolean(formik.errors.game_type)
                }
                defaultValue={GameType.sixes}
            >
                {GameTypes.map((v) => (
                    <MenuItem key={`gameType-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.game_type && formik.errors.game_type}
            </FormHelperText>
        </FormControl>
    );
};
