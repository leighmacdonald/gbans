import {parseISO} from "date-fns";
import { expect } from 'chai';

test('Parses go date format', () => {
    expect(parseISO("2021-03-02T08:17:55.015224Z").toISOString())
        .to.equal("2021-03-02T08:17:55.015Z", "Invalid date parsed")
});