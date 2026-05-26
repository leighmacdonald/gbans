import type { ReportStatus } from "../../../rpc/ban/v1/report_pb";
import SelectField from "./SelectField";

export const ReportStatusField = SelectField<ReportStatus>;

export default ReportStatusField;
