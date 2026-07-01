import type { Bucket } from "../../../rpc/stats/v1/stats_pb";
import SelectField from "./SelectField";

export const BucketField = SelectField<Bucket>;

export default BucketField;
