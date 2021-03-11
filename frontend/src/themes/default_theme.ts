import {createMuiTheme} from "@material-ui/core";
import {red} from "@material-ui/core/colors";

const default_theme = createMuiTheme({
    palette: {
        primary: {
            main: '#556cd6',
        },
        secondary: {
            main: '#19857b',
        },
        error: {
            main: red.A400,
        },
        background: {
            default: '#fff',
        },
    },
})

export default default_theme;