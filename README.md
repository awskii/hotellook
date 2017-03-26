## Hotellook API

Simple implementation of [this](https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей) API. 

Make sure you have set **valid** marker and token before running tests.

If you don't have a token, you should pass `searchID = -1` to `FetchSearchResults` - library will use real saved response from server.

### Installation:
`go get github.com/awskii/hotellook`

### Docs:
[GoDoc](https://godoc.org/github.com/awskii/hotellook) or run

`$ godoc github.com/awskii/hotellook`

### Example:

Getting hotels in Saint-Petersburg, Russia
```go
package main

import (
    "log"
    "github.com/awskii/hotellook"
)

const (
    marker = 1234
    token = "your_api_token"
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
    if len(res.Results.Location) == 0 {
        log.Println("City not found")
        return
    }

    cityId := res.Results.Location[0].ID // 12196 = Saint-Petersburg, Russia 

    searchRequest := &hotellook.SearchRequest{
        CityID:        cityId,
        CheckIn:       "2016-12-31",
        CheckOut:      "2017-01-02",
        AdultsCount:   1,
        ChildrenCount: 0,
        Lang:          "en",
        Currency:      "usd",
    }
    searchID, err := hotellook.Search(searchRequest)
    if err != nil {
        log.Fatalln(err.Error())
    }

    resp, err := hl.FetchSearchResults(&hotellook.SearchResultsRequest{
        SearchID: searchID,
        SortBy:   "price",
        SortAsc:  1,
    })
    if err != nil {
        log.Fatalln(err.Error())
    }
    for _, h := range resp.Results{
        ...
    }

}
```

