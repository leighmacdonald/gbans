import type { TimeBucket } from "../../../rpc/stats/v1/stats_pb";
import SelectField from "./SelectField";

export const StatsTimeBucketField = SelectField<TimeBucket>;

export default StatsTimeBucketField;
