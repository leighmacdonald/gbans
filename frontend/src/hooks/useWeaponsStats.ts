import { useEffect, useMemo, useState } from 'react';
import { apiGetPlayerWeaponStats, PlayerWeaponStats, QueryFilter, Weapon } from '../api';
import { logErr } from '../util/errors';
import { compare, RowsPerPage, stableSort } from '../util/table.ts';

export const useWeaponsStats = (weapon_id: number, opts: QueryFilter<PlayerWeaponStats>) => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [allStats, setAllStats] = useState<PlayerWeaponStats[]>([]);
    const [weapon, setWeapon] = useState<Weapon>();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        if (weapon_id <= 0) {
            return;
        }
        apiGetPlayerWeaponStats(weapon_id)
            .then((d) => {
                setAllStats(d.data);
                setCount(d.count);
                setWeapon(d.weapon);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
        return abortController.abort();
    }, [weapon_id]);

    const data = useMemo(() => {
        const limit = opts.limit ?? RowsPerPage.TwentyFive;
        const offset = opts.offset ?? 0;
        return stableSort(allStats, compare(opts.desc ? 'desc' : 'asc', opts.order_by ?? 'kills')).slice(offset, offset + limit);
    }, [allStats, opts.desc, opts.limit, opts.offset, opts.order_by]);

    return { data, weapon, count, loading };
};
