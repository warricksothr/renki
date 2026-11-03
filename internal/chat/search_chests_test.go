package chat

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestChestSearchCommandsRecognizeNearbyRadiusAndExplicitLists(t *testing.T) {
	for _, tc := range []struct {
		text string
		want ChestSearchRequest
	}{
		{"look for wood in a nearby chest", ChestSearchRequest{Item: "wood", Radius: 25}},
		{"search nearby chests for fine wood within 60m", ChestSearchRequest{Item: "fine wood", Radius: 60}},
		{"check all nearby chests for coal within 80 meters", ChestSearchRequest{Item: "coal", Radius: 80}},
		{"search chests 1:2, 1:3 for wood within 40m", ChestSearchRequest{Item: "wood", Radius: 40, Chests: []string{"1:2", "1:3"}}},
	} {
		b, f, _ := newBrain(nil)
		var got ChestSearchRequest
		b.opts.SearchChests = func(_ context.Context, by string, p ChestSearchRequest) (string, error) {
			if by != "Sothr" {
				t.Fatal(by)
			}
			got = p
			return "Searching without taking items.", nil
		}
		b.Chat(line("Sothr", "Bjorn, "+tc.text))
		if !reflect.DeepEqual(got, tc.want) || len(f.tasks()) != 0 {
			t.Fatal(tc.text, got, f.tasks())
		}
	}
}

func TestChestSearchAuthorizesBeforeStartingAndExplainsUnavailableOrBusy(t *testing.T) {
	for _, mode := range []string{"stranger", "unavailable", "retreat", "radius"} {
		t.Run(mode, func(t *testing.T) {
			b, f, _ := newBrain(nil)
			calls := 0
			b.opts.SearchChests = func(context.Context, string, ChestSearchRequest) (string, error) {
				calls++
				return "", errors.New("stop my current task first")
			}
			if mode == "unavailable" {
				b.opts.SearchChests = nil
			}
			by := "Sothr"
			if mode == "stranger" {
				by = "Mallory"
			}
			command := "look for wood in a nearby chest"
			if mode == "radius" {
				command += " within 900m"
			}
			b.Chat(line(by, "Bjorn "+command))
			want := 0
			if mode == "retreat" {
				want = 1
			}
			if calls != want || len(f.tasks()) != 0 || strings.Join(f.said(), " ") == "" {
				t.Fatal(calls, f.tasks(), f.said())
			}
		})
	}
}
