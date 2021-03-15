import * as React from 'react';
import { SyntheticEvent, useEffect } from 'react';
import IPCIDR from 'ip-cidr';
import {
    apiCreateBan,
    apiGetProfile,
    BanPayload,
    PlayerProfile
} from '../util/api';
import { Nullable } from '../util/types';

import {
    Button,
    createStyles,
    FormControl,
    FormControlLabel,
    FormHelperText,
    FormLabel,
    Grid,
    InputLabel,
    MenuItem,
    Radio,
    RadioGroup,
    Select,
    TextField,
    Typography
} from '@material-ui/core';
import { VoiceOverOffSharp } from '@material-ui/icons';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { log } from '../util/errors';

export const ip2int = (ip: string): number => {
    return (
        ip.split('.').reduce((ipInt, octet) => {
            return (ipInt << 8) + parseInt(octet, 10);
        }, 0) >>> 0
    );
};

export type BanType = 'network' | 'steam';

const useStyles = makeStyles((theme: Theme) =>
    createStyles({
        formControl: {
            margin: theme.spacing(1),
            minWidth: 120
        },
        selectEmpty: {
            marginTop: theme.spacing(2)
        }
    })
);

export enum Duration {
    dur15m = '15m',
    dur6h = '6h',
    dur12h = '12h',
    dur24h = '24h',
    dur48h = '48h',
    dur72h = '72h',
    dur1w = '1w',
    dur2w = '2w',
    dur1M = '1M',
    dur6M = '6M',
    dur1y = '1y',
    durInf = '∞',
    durCustom = 'custom'
}

const Durations = [
    Duration.dur15m,
    Duration.dur6h,
    Duration.dur12h,
    Duration.dur24h,
    Duration.dur48h,
    Duration.dur72h,
    Duration.dur1w,
    Duration.dur2w,
    Duration.dur1M,
    Duration.dur6M,
    Duration.dur1y,
    Duration.durInf,
    Duration.durCustom
];

export const PlayerBanForm = (): JSX.Element => {
    const classes = useStyles();

    const [fSteam, setFSteam] = React.useState<string>(
        'https://steamcommunity.com/id/SQUIRRELLY/'
    );
    const [duration, setDuration] = React.useState<Duration>(Duration.dur48h);
    const [reasonText, setReasonText] = React.useState<string>('');
    const [network, setNetwork] = React.useState<string>('');
    const [networkSize, setNetworkSize] = React.useState<number>(0);
    const [banType, setBanType] = React.useState<BanType>('steam');
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>();
    const loadPlayerSummary = async () => {
        try {
            setProfile((await apiGetProfile(fSteam)) as PlayerProfile);
        } catch (e) {
            log(e);
        }
    };
    useEffect(() => {
        // Validate results
    }, [profile]);
    const handleUpdateFSteam = React.useCallback(loadPlayerSummary, [
        setProfile,
        fSteam
    ]);

    const handleSubmit = React.useCallback(async () => {
        if (!profile || profile?.player?.steam_id > 0) {
            return;
        }
        const opts: BanPayload = {
            steam_id: profile.player.steamid ?? '',
            ban_type: 2,
            duration: duration,
            network: banType === 'steam' ? '' : network,
            reason_text: reasonText,
            reason: 0
        };
        const r = await apiCreateBan(opts);
        log(`${r}`);
    }, [profile, reasonText, network, banType, duration]);

    const handleUpdateNetwork = (evt: SyntheticEvent) => {
        const value = (evt.target as HTMLInputElement).value;
        setNetwork(value);
        if (value !== '') {
            try {
                const cidr = new IPCIDR(value);
                if (cidr != undefined) {
                    setNetworkSize(
                        ip2int(cidr?.end()) - ip2int(cidr?.start()) + 1
                    );
                }
            } catch (e) {
                if (e instanceof TypeError) {
                    // TypeError on invalid input we can ignore
                } else {
                    throw e;
                }
            }
        }
    };

    const handleUpdateReasonText = (evt: SyntheticEvent) => {
        setReasonText((evt.target as HTMLInputElement).value);
    };

    const handleUpdateDuration = (
        evt: React.ChangeEvent<{ value: unknown }>
    ) => {
        setDuration((evt.target.value as Duration) ?? Duration.durInf);
    };

    const onChangeFStream = (evt: React.ChangeEvent<HTMLInputElement>) => {
        setFSteam((evt.target as HTMLInputElement).value);
    };

    const onChangeType = (evt: React.ChangeEvent<HTMLInputElement>) => {
        setBanType(evt.target.value as BanType);
    };

    return (
        <Grid container spacing={3}>
            <Grid item xs={12}>
                <Typography variant={'h1'}>Ban A Player</Typography>
            </Grid>
            <form noValidate>
                <Grid container>
                    <Grid item xs>
                        <TextField
                            fullWidth
                            id={'query'}
                            label={'Steam ID / Profile URL'}
                            onChange={onChangeFStream}
                            onBlur={handleUpdateFSteam}
                        />
                    </Grid>
                </Grid>
                <Grid item xs={12}>
                    <FormControl component="fieldset">
                        <FormLabel component="legend">Ban Type</FormLabel>
                        <RadioGroup
                            aria-label="gender"
                            name="gender1"
                            value={fSteam}
                            onChange={onChangeType}
                            row
                        >
                            <FormControlLabel
                                value="steam"
                                control={<Radio />}
                                label="Steam"
                            />
                            <FormControlLabel
                                value="network"
                                control={<Radio />}
                                label="IP / Network"
                            />
                        </RadioGroup>
                    </FormControl>
                </Grid>
                {banType === 'network' && (
                    <>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth={true}
                                id={'network'}
                                label={'Network Range (CIDR Format)'}
                                onChange={handleUpdateNetwork}
                            />
                        </Grid>
                        <Grid item>
                            <Typography variant={'body1'}>
                                Current number of hosts in range: {networkSize}
                            </Typography>
                        </Grid>
                    </>
                )}
                <Grid item>
                    <TextField
                        fullWidth
                        id={'duration'}
                        label={'Network Range (CIDR Format)'}
                        onChange={handleUpdateNetwork}
                    />
                </Grid>
                <Grid item>
                    <TextField
                        fullWidth
                        id={'reason'}
                        label={'Ban Reason'}
                        onChange={handleUpdateReasonText}
                    />
                </Grid>
                <Grid item>
                    <FormControl className={classes.formControl}>
                        <InputLabel id="demo-simple-select-helper-label">
                            Age
                        </InputLabel>
                        <Select
                            fullWidth
                            labelId="demo-simple-select-helper-label"
                            id="demo-simple-select-helper"
                            value={duration}
                            onChange={handleUpdateDuration}
                        >
                            {Durations.map((v) => (
                                <MenuItem key={`time-${v}`} value={v}>
                                    {v}
                                </MenuItem>
                            ))}
                        </Select>
                        <FormHelperText>
                            Some important helper text
                        </FormHelperText>
                    </FormControl>
                </Grid>
                <Grid item xs={12}>
                    <Button
                        fullWidth
                        key={'submit'}
                        value={'Create Ban'}
                        onClick={handleSubmit}
                        startIcon={<VoiceOverOffSharp />}
                    >
                        Ban Player
                    </Button>
                </Grid>
            </form>
        </Grid>
    );
};

// function x() {
//     return <div className={"grid-x grid-padding-x"}>
//         <div className={"cell medium-6"}>
//             <div className={"grid-y"}>
//                 <h2>Ban Details</h2>
//             </div>
//             <div className={"grid-y"}>
//                 <div className={"cell"}>
//                     <label form={"fSteam"}>Steam ID / Profile URL</label>
//                 </div>
//                 <div className={"cell"}>
//                     <input name={"fSteam"} type={"text"} value={fSteam} onChange={onChangeFStream}
//                            onBlur={handleUpdateFSteam}/>
//                 </div>
//                 <div className={"cell"}>
//                     <div className={"grid-x"}>
//                         <fieldset className="cell">
//                             <legend>Ban Mode</legend>
//                             <p>SteamID is optional for network bans, however it can be used to trace the
//                                 initial culprit of a ban.
//                             </p>
//                             <input type={"radio"} name={"ban_type"} value={"steam"}
//                                    checked={banType == "steam"} id={"steam"} onChange={(() => {
//                                 setBanType("steam")
//                             })}/><label htmlFor={"steam"}>Steam Ban</label>
//                             <input type={"radio"} name={"ban_type"} value={"network"}
//                                    checked={banType == "network"} id={"network"} onChange={() => {
//                                 setBanType("network")
//                             }}/><label htmlFor={"network"}>Network Ban</label>
//                         </fieldset>
//                     </div>
//                 </div>
//                 {banType == "network" && <>
//                     <div className={"cell"}>
//                         <label form={"fSteam"}>Network Range (CIDR Format)</label>
//                     </div>
//                     <div className={"cell"}>
//                         <input name={"network"} type={"text"} value={network} placeholder={"12.34.56.78/32"}
//                                onChange={handleUpdateNetwork}
//                                title={"Must be CIDR format with 2 digit mask"}
//                                pattern={"^(\\d{1,3}[\\.\\/]){4}\\d{2}$"}
//                         />
//                         <p>Current number of hosts in range: {networkSize}</p>
//                     </div>
//                 </>
//                 }
//                 <div className={"cell"}>
//                     <label form={"fSteam"}>Reason</label>
//                 </div>
//                 <div className={"cell"}>
//                     <input name={"reason_text"} type={"text"} value={reasonText}
//                            onChange={handleUpdateReasonText}/>
//                 </div>
//
//                 <div className={"cell"}>
//                     <label form={"duration"}>Duration</label>
//                 </div>
//                 <div className={"cell"}>
//                     <select onChange={handleUpdateDuration} value={duration}>
//                         {["15m", "6h", "12h", "24h", "48h", "72h", "1w", "2w", "1m", "6m", "1y", "∞", "custom"].map((v) => {
//                                 return <option key={`time-${v}`} value={v}>{v}</option>
//                             }
//                         )}
//                     </select>
//                     {duration === "custom" && (
//                         <label form={"duration"}>
//                             Custom Duration
//                             <input name={"duration"} type={"text"} placeholder={"5d"}/>
//                         </label>
//                     )}
//                 </div>
//                 <div className={"cell"}>
//                     <a className={"button"} onClick={handleSubmit}>Submit Ban <i className={"fi-flag"}
//                                                                                  style={{"color": "red"}}/></a>
//                 </div>
//             </div>
//         </div>

//         <div className={"cell medium-6"}>
//             {profile?.player && profile?.player?.avatarfull &&
//             <div className={"grid-y"}>
//                 <div className={"cell"}>
//                     <div className="expanded button-group">
//                         <a className={!friendsPage ? "button" : "button secondary"} onClick={() => {
//                             setShowFriends(false);
//                         }}>Profile</a>
//                         <a className={friendsPage ? "button" : "button secondary"} onClick={() => {
//                             setShowFriends(true);
//                         }}>Friends ({profile?.friends?.length ?? "n/a"})</a>
//                     </div>
//                 </div>
//                 {!showFriends && <>
//                     <div className={"cell"}>
//                         <figure className={"text-center"}>
//                             <img src={profile.player.avatarfull} alt={"Current user avatar"}/>
//                             <figcaption>{profile.player.steamid}</figcaption>
//                         </figure>
//                     </div>
//                     <div className={"cell"}>
//                         <h4 className={"text-center"}>{profile.player.personaname}</h4>
//                         {profile.player.realname != "" && (
//                             <h4 className={"text-center"}><small>{profile.player.realname}</small></h4>
//                         )}
//                     </div>
//                     <div className={"cell"}>
//                         <dl>
//                             <dt>Community Visibility State</dt>
//                             <dd>{profile.player.communityvisibilitystate == communityVisibilityState.Public ? "Public" : "Private"}</dd>
//
//                             <dt>Profile State</dt>
//                             <dd>{profile.player.profilestate ? "Configured" : "Non-Configured"}</dd>
//                             {profile.player?.timecreated && <>
//                                 <dt>Created</dt>
//                                 <dd>{format(fromUnixTime(profile.player.timecreated), "dd-MM-Y")}</dd>
//                                 <dt>Age</dt>
//                                 <dd>{formatDistanceToNow(fromUnixTime(profile.player.timecreated), {
//                                     addSuffix: false,
//                                     includeSeconds: true
//                                 })}</dd>
//                             </>}
//                             {profile.player.loccountrycode != "" && <>
//                                 <dt>Country</dt>
//                                 <dd>{profile.player.loccountrycode}</dd>
//                             </>}
//                             {profile.player.locstatecode != "" && <>
//                                 <dt>State/Province</dt>
//                                 <dd>{profile.player.locstatecode}</dd>
//                             </>}
//                         </dl>
//                     </div>
//                 </>}
//                 {showFriends &&
//                 <>
//                     <div className="expanded button-group">
//                         {chunk(profile.friends, 25).map((_, index) => {
//                             return (
//                                 <a key={`button-${index}`}
//                                    className={index == friendsPage ? "button" : "button secondary"}
//                                    onClick={(() => {
//                                        setFriendsPage(index)
//                                    })}>{index}</a>
//                             )
//                         })}
//                     </div>
//                     <div className="grid-y grid-padding-y">
//                         {profile.friends.filter((_, index) => {
//                             return index + 1 >= Math.max(friendsPage, 0) * 25 && index + 1 <= Math.max(friendsPage + 1, 0) * 25
//                         }).map((value) => {
//                             return <div className={"grid-x grid-padding-x"} key={`friend-${value.steamid}`}>
//                                 <a className={"cell"} target={"_blank"} style={{"display": "inline-block"}}
//                                    href={value.profileurl}>
//                                     <img src={value.avatar} alt={"Profile Avatar"}/> {value.personaname}
//                                 </a>
//                             </div>
//                         })}
//                     </div>
//                 </>}
//             </div>
//             }
//         </div>
//     </div>
// </Grid>
// </Grid>
// )
// }
