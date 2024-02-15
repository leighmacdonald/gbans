import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export enum GameConfig {
    etf2l = 'etf2l',
    rgl = 'rgl',
    ugc = 'ugc',
    ozfortress = 'ozfortress'
}

export const GameConfigs = [
    GameConfig.etf2l,
    GameConfig.rgl,
    GameConfig.ugc,
    GameConfig.ozfortress
];

export const gameConfigValidator = yup
    .string()
    .test('checkGameConfig', 'Invalid game type selection', (gameConfig) =>
        GameConfigs.includes(gameConfig as GameConfig)
    )
    .label('Select a config to use')
    .required('game config is required');

export const GameConfigField = ({
    formik
}: {
    formik: FormikState<{
        game_config: GameConfig;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="actionType-label">Game Config</InputLabel>
            <Select<GameConfig>
                fullWidth
                disabled={formik.isSubmitting}
                label={'Game Config'}
                labelId="game_config-label"
                id="game_config"
                name={'game_config'}
                value={formik.values.game_config}
                onChange={formik.handleChange}
                error={
                    formik.touched.game_config &&
                    Boolean(formik.errors.game_config)
                }
                defaultValue={GameConfig.rgl}
            >
                {GameConfigs.map((v) => (
                    <MenuItem key={`game_config-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.game_config && formik.errors.game_config}
            </FormHelperText>
        </FormControl>
    );
};
