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

var ErrBind = errors.New("bind error")

// Bind is responsible for binding the input values from a discord modal input into a struct
// of the type T. Fields are mapped with the `id` field tag which has a int value which corresponds
// to a unique `Component.ID` for each user input field in the modal.
func Bind[T any](ctx context.Context, interaction *discordgo.InteractionCreate) (T, error) {
	values, errValues := mapInteractionValues(interaction)
	if errValues != nil {
		var value T

		return value, errValues
	}

	return bindValues[T](ctx, values)
}

// mapInteractionValues is responsible for transforming the interaction values into a map indexed by the unique
// input ID integer.
func mapInteractionValues(interaction *discordgo.InteractionCreate) (map[int]string, error) {
	values := map[int]string{}

	// Parse modal data into values map
	for _, component := range interaction.ModalSubmitData().Components {
		switch component.Type() {
		case discordgo.ActionsRowComponent:
			row := component.(*discordgo.ActionsRow)
			for _, comp := range row.Components {
				switch comp.Type() {
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
			row := component.(*discordgo.Label)
			comp := row.Component.(*discordgo.SelectMenu)
			if len(comp.Values) > 0 {
				values[comp.ID] = comp.Values[0]
			}
		}
	}

	return values, nil
}

func bindValues[T any](ctx context.Context, values map[int]string) (T, error) {
	var value T
	// Use reflection to populate struct fields based on `id` tags
	elem := reflect.ValueOf(&value).Elem()
	elemType := elem.Type()

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		fieldValue := elem.Field(i)

		// Get the `id` tag
		idTag := field.Tag.Get("id")
		if idTag == "" {
			continue
		}

		// Parse the id tag as an integer
		fieldID, errParse := strconv.Atoi(idTag)
		if errParse != nil {
			return value, fmt.Errorf("invalid id tag on field %s: %w", field.Name, errParse)
		}

		val, exists := values[fieldID]
		if !exists {
			continue
		}

		if !fieldValue.CanSet() {
			continue
		}

		switch fieldValue.Interface().(type) {
		case *netip.Prefix:
			if val == "" {
				continue
			}
			prefix, prefixErr := netip.ParsePrefix(val)
			if prefixErr != nil {
				return value, prefixErr
			}
			fieldValue.Set(reflect.ValueOf(&prefix))
		case *duration.Duration:
			durVal, errDur := duration.Parse(val)
			if errDur != nil {
				return value, errDur
			}
			fieldValue.Set(reflect.ValueOf(durVal))
		case steamid.SteamID:
			sid, errResolve := steamid.Resolve(ctx, val)
			if errResolve != nil {
				return value, errResolve
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
			if reflect.TypeOf(val).AssignableTo(fieldValue.Type()) {
				fieldValue.Set(reflect.ValueOf(val))
			}
		default:
			return value, fmt.Errorf("%w: unahndled type: %s", ErrBind, field.Name)
		}
	}

	return value, nil
}
