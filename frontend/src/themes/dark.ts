import createTheme from '@mui/material/styles/createTheme';
import darkScrollbar from '@mui/material/darkScrollbar';

const darkTheme = createTheme({
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: darkScrollbar()
            }
        }
    },
    typography: {
        fontSize: 12,
        h1: {
            fontSize: 36
        },
        h2: {
            fontSize: 32
        },
        h3: {
            fontSize: 28
        },
        h4: {
            fontSize: 24
        },
        h5: {
            fontSize: 20
        },
        h6: {
            fontSize: 16
        }
    },
    palette: {
        mode: 'dark',
        // primary: {
        //     main: '#943b00'
        // },
        // secondary: {
        //     main: '#836312'
        // },
        // error: {
        //     main: '#8d0101'
        // },
        background: {
            default: '#363636'
        }
    }
});

export default darkTheme;
