import { type HTMLProps, useEffect, useRef } from "react";

export function IndeterminateCheckbox({
	indeterminate,
	...rest
}: { indeterminate?: boolean } & HTMLProps<HTMLInputElement>) {
	const ref = useRef<HTMLInputElement>(null!);

	useEffect(() => {
		if (typeof indeterminate === "boolean") {
			ref.current.indeterminate = !rest.checked && indeterminate;
		}
	}, [indeterminate, rest.checked]);

	return <input type="checkbox" ref={ref} {...rest} />;
}
