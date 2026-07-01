import type { Bucket } from "../../../rpc/stats/v1/stats_pb";
import SelectField from "./SelectField";

export const SelectBucketField = SelectField<Bucket>;

export default SelectBucketField;
