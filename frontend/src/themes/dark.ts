import { createMuiTheme } from '@material-ui/core';

const darkTheme = createMuiTheme({
    typography: {
        fontSize: 12,
        h1: {
            fontSize: 32
        },
        h2: {
            fontSize: 28
        },
        h3: {
            fontSize: 24
        },
        h4: {
            fontSize: 20
        },
        h5: {
            fontSize: 16
        },
        h6: {
            fontSize: 14
        }
    },
    palette: {
        type: 'dark',
        primary: {
            main: '#943b00'
        },
        secondary: {
            main: '#836312'
        },
        error: {
            main: '#8d0101'
        },
        background: {
            default: '#1c1c1c'
        }
    }
});

export default darkTheme;
