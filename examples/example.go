package main

import (
	"fmt"
	"github.com/awskii/hotellook"
	"log"
)

const (
	marker = 1234
	token  = "your_api_token"
)

func main() {
	hl := hotellook.NewAPI(marker)
	hl.SetToken(token)

	lookupReq := &hotellook.LookupRequest{
		Query:   "Saint-Petersburg",
		Lang:    "en",
		LookFor: "both",
	}

	// Asking meta information about city (location, city ID and so on).
	res, err := hl.Lookup(lookupReq)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if len(res.Results.Locations) == 0 {
		log.Println("City not found")
		return
	}

	log.Println(res.Results.Locations[0].ID, res.Results.Locations[0].CityName, res.Results.Locations[0].Iata)

	// As far as we using invalid token and markder,
	// we don't need to use that method.
	// searchRequest := &hotellook.SearchRequest{
	//     CityID:        cityId,
	//     CheckIn:       "2016-12-31",
	//     CheckOut:      "2017-01-02",
	//     AdultsCount:   1,
	//     ChildrenCount: 0,
	//     Lang:          "en",
	//     Currency:      "usd",
	// }
	// searchID, err := hotellook.Search(searchRequest)
	// if err != nil {
	//     log.Fatalln(err.Error())
	// }

	resp, err := hl.FetchSearchResults(&hotellook.SearchResultsRequest{
		SearchID: -1,
		SortBy:   "price",
		SortAsc:  1,
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
	for _, h := range resp.Results {
		fmt.Println(h.GuestScore)
	}
}
