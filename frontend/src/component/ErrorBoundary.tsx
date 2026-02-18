import React from "react";

export const errorDialog = ({
	error,
	componentStack,
}: {
	error: unknown;
	componentStack: string;
	eventId: string;
	resetError(): void;
}) => {
	return (
		<React.Fragment>
			<div>You have encountered an error</div>
			<div>{error as string}</div>
			<div>{componentStack}</div>
		</React.Fragment>
	);
};
