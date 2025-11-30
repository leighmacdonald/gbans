package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"reflect"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

// ModalLabelID defines the value used for both the unique ID value of a
// discordgo.MessageComponent ID field and the subsequent mapping of its value to a struct
// field with the matching tagName struct tag value set.
type ModalLabelID int

const (
	IDSteamID ModalLabelID = iota + 1
	IDCIDR
	IDReason
	IDDuration
	IDNotes
	IDBody
	IDImmunityLevel
	IDAlias
	IDFlags
	IDGroupID
)

const tagName = "id"

var ErrBind = errors.New("bind error")

// Bind is responsible for binding the input values from a discord modal input into a struct
// of the type T. Fields are mapped with the `id` field tag which has a int value which corresponds
// to a unique `Component.ID` for each user input field in the modal.
func Bind[T any](ctx context.Context, components []discordgo.MessageComponent) (T, error) { //nolint:ireturn
	values, errValues := mapInteractionValues(components)
	if errValues != nil {
		var value T

		return value, errValues
	}
	return BindValues[T](ctx, values)
}

// mapInteractionValues is responsible for transforming the interaction values into a map indexed by the unique
// input ID integer.
func mapInteractionValues(components []discordgo.MessageComponent) (map[int]string, error) {
	values := map[int]string{}
	// e := buildModalValueMap(values, components...)
	// return values, e

	// Parse modal data into values map
	for _, component := range components {
		switch component.Type() {
		case discordgo.ActionsRowComponent:
			row, castOK := component.(*discordgo.ActionsRow)
			if !castOK {
				continue
			}
			for _, comp := range row.Components {
				switch comp.Type() { //nolint:gocritic
				case discordgo.TextInputComponent:
					choice, ok := comp.(*discordgo.TextInput)
					if !ok {
						slog.Error("Failed to cast to textinput")

						return values, nil
					}
					values[choice.ID] = choice.Value
				}
			}
		case discordgo.LabelComponent:
			row, castOK := component.(*discordgo.Label)
			if !castOK {
				continue
			}
			comp, castMenu := row.Component.(*discordgo.SelectMenu)
			if !castMenu {
				continue
			}
			if len(comp.Values) > 0 {
				values[comp.ID] = comp.Values[0]
			}
		}
	}

	return values, nil
}

// buildModalValueMap recursively finds all of the values of the components returned from a modal.
func buildModalValueMap(results map[int]string, components ...discordgo.MessageComponent) error {
	for _, component := range components {
		switch component.Type() {
		case discordgo.TextInputComponent:
			choice, ok := component.(*discordgo.TextInput)
			if !ok {
				return fmt.Errorf("%w: Failed to cast to textinput", ErrBind)
			}
			results[choice.ID] = choice.Value
		case discordgo.SelectMenuComponent:
			selectMenu, castMenu := component.(*discordgo.SelectMenu)
			if !castMenu {
				return fmt.Errorf("%w: Failed to cast to SelectMenu", ErrBind)
			}
			if len(selectMenu.Values) > 0 {
				results[selectMenu.ID] = selectMenu.Values[0]
			}
		case discordgo.SectionComponent:
			section, castMenu := component.(*discordgo.Section)
			if !castMenu {
				return fmt.Errorf("%w: Failed to cast to Section", ErrBind)
			}

			return buildModalValueMap(results, section.Components...)
		case discordgo.ContainerComponent:
			container, castMenu := component.(*discordgo.Container)
			if !castMenu {
				return fmt.Errorf("%w: Failed to cast to Container", ErrBind)
			}

			return buildModalValueMap(results, container.Components...)
		case discordgo.LabelComponent:
			label, castMenu := component.(*discordgo.Label)
			if !castMenu {
				return fmt.Errorf("%w: Failed to cast to Label", ErrBind)
			}

			return buildModalValueMap(results, label.Component)
		default:
			continue
		}
	}

	return nil
}

// BindValues handles mapping the input options into the T response type. It Uses reflection to populate struct fields
// based on `id` tags and their values defined using ModalLabelID.
func BindValues[T any](ctx context.Context, values map[int]string) (T, error) { //nolint:ireturn
	var (
		value    T
		elem     = reflect.ValueOf(&value).Elem()
		elemType = elem.Type()
	)

	for i := range elemType.NumField() {
		field := elemType.Field(i)
		fieldValue := elem.Field(i)

		// Get the tag value
		idTag := field.Tag.Get(tagName)
		if idTag == "" {
			continue
		}

		// Parse the id tag as an integer
		fieldID, errParse := strconv.Atoi(idTag)
		if errParse != nil {
			return value, fmt.Errorf("invalid %s tag on field %s: %w", idTag, field.Name, errParse)
		}

		val, exists := values[fieldID]
		if !exists || val == "" || !fieldValue.CanSet() {
			continue
		}

		switch fieldValue.Interface().(type) {
		case *netip.Prefix:
			prefix, prefixErr := netip.ParsePrefix(val)
			if prefixErr != nil {
				return value, fmt.Errorf("%w: %w: %s:%s", ErrBind, prefixErr, field.Name, val)
			}
			fieldValue.Set(reflect.ValueOf(&prefix))
		case *duration.Duration:
			durVal, errDur := duration.Parse(val)
			if errDur != nil {
				return value, fmt.Errorf("%w: %w: %s:%s", ErrBind, errDur, field.Name, val)
			}
			fieldValue.Set(reflect.ValueOf(durVal))
		case steamid.SteamID:
			sid, errResolve := steamid.Resolve(ctx, val)
			if errResolve != nil {
				return value, fmt.Errorf("%w: %w: %s:%s", ErrBind, errResolve, field.Name, val)
			}
			if !sid.Valid() {
				return value, fmt.Errorf("%w: invalid steamid tag on field %s: %s", ErrBind, field.Name, val)
			}
			fieldValue.Set(reflect.ValueOf(sid))
		case reason.Reason:
			intVal, errVal := strconv.Atoi(val)
			if errVal != nil {
				return value, fmt.Errorf("%w: invalid reason tag on field %s: %w", ErrBind, field.Name, errVal)
			}
			fieldValue.Set(reflect.ValueOf(reason.Reason(intVal)))
		case string:
			if reflect.TypeOf(val).AssignableTo(fieldValue.Type()) { // nolint:modernize
				fieldValue.Set(reflect.ValueOf(val))
			}
		case int:
			intVal, errVal := strconv.Atoi(val)
			if errVal != nil {
				return value, fmt.Errorf("%w: invalid reason tag on field %s: %w", ErrBind, field.Name, errVal)
			}
			fieldValue.Set(reflect.ValueOf(intVal))
		case int64:
			intVal, errVal := strconv.ParseInt(val, 10, 64)
			if errVal != nil {
				return value, fmt.Errorf("%w: invalid reason tag on field %s: %w", ErrBind, field.Name, errVal)
			}
			fieldValue.Set(reflect.ValueOf(intVal))
		default:
			return value, fmt.Errorf("%w: unhandled type: %s", ErrBind, field.Name)
		}
	}

	return value, nil
}
