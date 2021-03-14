import {createMuiTheme} from "@material-ui/core";
import {red} from "@material-ui/core/colors";

const darkTheme = createMuiTheme({
    palette: {
        type: "dark",
        primary: {
            main: '#121b47',
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

export default darkTheme;