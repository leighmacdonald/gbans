import React, {useState} from "react";
import {Button, createStyles, Grid, TextField} from "@material-ui/core";
import {makeStyles, Theme} from "@material-ui/core/styles";

const useStyles = makeStyles((theme: Theme) =>
    createStyles({
        root: {
            '& > *': {
                margin: theme.spacing(1),
                width: '25ch',
            },
        },
    }),
);

export const ServerAddForm = () => {
    const classes = useStyles();
    const [name, setName] = useState<string>("")
    const [address, setAddress] = useState<string>("")
    const [port, setPort] = useState<number>(27015)

    return (
        <Grid container>
            <form className={classes.root} noValidate autoComplete="off">
                <Grid item xs={12}>
                    <TextField id="standard-basic" label="Short Server Name (Example: eg-1)" fullWidth value={name} onChange={(v) => {
                        setName(v.target.value)
                    }}/>
                </Grid>
                <Grid item xs={12}>
                    <TextField id="standard-basic" label="Hostname or IP" fullWidth value={address} onChange={(v) => {
                        setAddress(v.target.value)
                    }}/>
                </Grid>
                <Grid item xs={12}>
                    <TextField type="number" id="standard-basic" label="Port (Default: 27015)" fullWidth value={port} onChange={(v) => {
                        setPort(parseInt(v.target.value))
                    }}/>
                </Grid>
                <Grid item xs={12}>
                    <Button variant={"contained"} type={"submit"} >Add Server</Button>
                </Grid>
            </form>
        </Grid>
    )
}