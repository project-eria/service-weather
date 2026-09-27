package main

import "testing"

func TestParse(t *testing.T) {
	body := []byte(`{"current":{"time":"2026-09-27T09:45","interval":900,"temperature_2m":17.6,
		"relative_humidity_2m":72,"pressure_msl":1018.6,"wind_speed_10m":2.01,
		"wind_direction_10m":153,"uv_index":null}}`)
	got, err := parse(body)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"temperature": 17.6, "humidity": 72, "pressure": 1019, "windStrength": 2.01, "windAngle": 153,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s: got %#v, want %#v", k, got[k], v)
		}
	}
	if _, err := parse([]byte(`{"error":true}`)); err == nil {
		t.Fatal("expected an error without current data")
	}
}
