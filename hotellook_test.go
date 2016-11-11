package hotellook

import (
	"strings"
	"testing"
)

const (
	token  = "bqadagadoqadjmcocciox1grdvp3ag"
	marker = 35290

	validMarker = 77777
	validToken  = "YOUR_APPROVED_TOKEN"
)

func TestNewAPI(t *testing.T) {
	api := NewAPI(marker)
	if api == nil {
		t.Fatal("NewAPI returns nil result with correct makrer")
	}
	api = NewAPI(0)
	if api != nil {
		t.Fatal("NewAPI returns non-nil result with marker=0")
	}
}

func TestCheckAccess(t *testing.T) {
	api := NewAPI(marker)
	api.SetToken("")
	if err := api.checkAccess(); err != ErrNoAccess {
		t.Fatal("Invalid result of checkAccess with empty token")
	}
}

func TestWithSignature(t *testing.T) {
	api := NewAPI(marker)
	api.SetToken(token)
	encoded := api.withSignature(nil)
	if encoded != "marker=35290&signature=abdab6a981233bdaf156a5abc17cb382" {
		t.Error(encoded)
		t.Fatal("withSignature returns incorrect result with nil params")
	}

	v := make(map[string]string)
	v["query"] = "moscow"
	v["lang"] = "en"
	v["limit"] = "1"
	v["lookFor"] = "both"

	if !strings.Contains(api.withSignature(v), "7386867331120289e76303d286d1758b") {
		t.Fatal("withSignature returns " + api.withSignature(v) + ", expected 7386867331120289e76303d286d1758b")
	}
}

func TestLookup(t *testing.T) {
	api := NewAPI(marker)
	api.SetToken(token)
	_, err := api.Lookup(&LookupRequest{
		Query:   "moscow",
		Lang:    "ru",
		LookFor: "both",
		Limit:   2,
	})
	if err != nil {
		t.Fatal("got nil response")
	}
}

func TestPrice(t *testing.T) {
	api := NewAPI(marker)
	api.SetToken(token)
	_, err := api.Price(&PriceRequest{
		Location: "MOW",
		CheckIn:  "2016-12-10",
		CheckOut: "2016-12-17",
		Currency: "rub",
		Limit:    10,
	})
	if err != nil {
		t.Fatal("got nil response")
	}
}

func TestCountries(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.Countries(); err != nil {
		t.Fatal(err.Error())
		t.Fatal("invalid token")
	}
}

func TestCities(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.Cities(); err != nil {
		t.Fatal(err.Error())
		t.Fatal("invalid token")
	}
}

func TestAmenities(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.Amenities(); err != nil {
		t.Fatal(err.Error())
		t.Fatal("invalid token")
	}
}

func TestHotelList(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.FetchHotelList("895"); err != nil {
		t.Fatal(err.Error())
	}
}

func TestRoomTypes(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.RoomTypes(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestFetchSearchResults(t *testing.T) {
	api := NewAPI(validMarker)
	api.SetToken(validToken)
	if _, err := api.FetchSearchResults(&SearchResultsRequest{
		SearchID: "-1",
	}); err != nil {
		t.Fatal(err.Error())
	}

}
