import React from 'react';
import demoClassImg from '../icons/class_demoman.png';
import scoutClassImg from '../icons/class_scout.png';
import engineerClassImg from '../icons/class_engineer.png';
import pyroClassImg from '../icons/class_pyro.png';
import heavyClassImg from '../icons/class_heavy.png';
import sniperClassImg from '../icons/class_sniper.png';
import spyClassImg from '../icons/class_spy.png';
import soldierClassImg from '../icons/class_soldier.png';
import medicClassImg from '../icons/class_medic.png';
import { PlayerClass, PlayerClassNames } from '../api';

export interface PlayerClassImgProps {
    cls: PlayerClass;
    size?: number;
}

export const PlayerClassImg = ({ cls, size = 24 }: PlayerClassImgProps) => {
    switch (cls) {
        case PlayerClass.Demo:
            return (
                <img
                    src={demoClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Scout:
            return (
                <img
                    src={scoutClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Engineer:
            return (
                <img
                    src={engineerClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Pyro:
            return (
                <img
                    src={pyroClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Heavy:
            return (
                <img
                    src={heavyClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Sniper:
            return (
                <img
                    src={sniperClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Spy:
            return (
                <img
                    src={spyClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        case PlayerClass.Soldier:
            return (
                <img
                    src={soldierClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
        default:
            return (
                <img
                    src={medicClassImg}
                    alt={PlayerClassNames[cls]}
                    width={size}
                    height={size}
                />
            );
    }
};
