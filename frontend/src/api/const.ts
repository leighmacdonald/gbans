export enum Team {
    UNASSIGNED,
    SPEC,
    RED,
    BLU
}

export enum PlayerClass {
    Spectator,
    Scout,
    Soldier,
    Pyro,
    Demo,
    Heavy,
    Engineer,
    Medic,
    Sniper,
    Spy,
    Unknown
}

export const PlayerClassNames: Record<PlayerClass, string> = {
    [PlayerClass.Spectator]: 'spectator',
    [PlayerClass.Scout]: 'scout',
    [PlayerClass.Soldier]: 'soldier',
    [PlayerClass.Pyro]: 'pyro',
    [PlayerClass.Demo]: 'demo',
    [PlayerClass.Heavy]: 'heavy',
    [PlayerClass.Engineer]: 'engineer',
    [PlayerClass.Medic]: 'medic',
    [PlayerClass.Sniper]: 'sniper',
    [PlayerClass.Spy]: 'spy',
    [PlayerClass.Unknown]: 'unknown'
};

export const sessionKeyDemoName = 'demoName';
export const sessionKeyReportPersonMessageIdName = 'rpmid';
export const sessionKeyReportSteamID = 'report_steam_id';

export const EmptyUUID = 'feb4bf16-7f55-4cb4-923c-4de69a093b79';
