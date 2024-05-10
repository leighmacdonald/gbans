export const findSelectedRows = <T>(selection: object, array: T[]) => {
    try {
        const selectedIndices = Object.keys(selection).map(Number);
        return (
            array.filter((_, index) => {
                return selectedIndices.includes(index);
            }) ?? undefined
        );
    } catch {
        return undefined;
    }
};
