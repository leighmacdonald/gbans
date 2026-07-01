import type { Category } from "../../../rpc/forum/v1/forum_pb";
import SelectField from "./SelectField";

export const ForumCategoryField = SelectField<Category>;

export default ForumCategoryField;
