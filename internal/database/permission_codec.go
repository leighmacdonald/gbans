package database

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
)

var errInvalidPermission = errors.New("invalid permission value")

// permissionCodec maps the roles.v1.Permission Go enum to the database `permission` enum type,
// without which pgx falls back to encoding the int32 value and the database rejects it.
type permissionCodec struct{}

func (permissionCodec) FormatSupported(format int16) bool {
	return format == pgtype.TextFormatCode || format == pgtype.BinaryFormatCode
}

func (permissionCodec) PreferredFormat() int16 {
	return pgtype.TextFormatCode
}

func (permissionCodec) PlanEncode(_ *pgtype.Map, _ uint32, format int16, value any) pgtype.EncodePlan {
	switch format {
	case pgtype.TextFormatCode, pgtype.BinaryFormatCode:
		if _, ok := value.(rolesv1.Permission); ok {
			return permEncodePlan{}
		}
	}

	return nil
}

type permEncodePlan struct{}

func (permEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	perm, ok := value.(rolesv1.Permission)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", errInvalidPermission, value)
	}

	return append(buf, perm.String()...), nil
}

func (permissionCodec) PlanScan(_ *pgtype.Map, _ uint32, format int16, target any) pgtype.ScanPlan {
	switch format {
	case pgtype.TextFormatCode, pgtype.BinaryFormatCode:
		if _, ok := target.(*rolesv1.Permission); ok {
			return permScanPlan{}
		}
	}

	return nil
}

func (permissionCodec) DecodeDatabaseSQLValue(m *pgtype.Map, oid uint32, format int16, src []byte) (driver.Value, error) {
	return permissionCodec{}.DecodeValue(m, oid, format, src)
}

func (permissionCodec) DecodeValue(_ *pgtype.Map, _ uint32, _ int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil //nolint:nilnil
	}

	return string(src), nil
}

type permScanPlan struct{}

func (permScanPlan) Scan(src []byte, dst any) error {
	perm, isPerm := dst.(*rolesv1.Permission)
	if !isPerm {
		return fmt.Errorf("%w: destination must be *roles.v1.Permission, got %T", errInvalidPermission, dst)
	}

	value, known := rolesv1.Permission_value[string(src)]
	if !known {
		return fmt.Errorf("%w: %q", errInvalidPermission, string(src))
	}

	*perm = rolesv1.Permission(value)

	return nil
}
