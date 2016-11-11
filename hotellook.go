// Implements travelpayouts API for hotel searches.
// Some methods require valid marker and token.
// Some methods returns static data, so feel free to cache their response.
//  [Cities, Countries, Amenities, HotelList, RoomTypes, HotelTypes, Photos]
package hotellook

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pquerna/ffjson/ffjson"
)

const apiURL = "http://engine.hotellook.com/api/v2/"

var (
	ErrNoAccess      = errors.New("You should specify valid token and marker to use this method")
	ErrEmptySearchID = errors.New("Empty search ID")
	ErrMissingParams = errors.New("Missing required parameters")
)

type API struct {
	token  string
	marker int

	mu      sync.Mutex
	remains int
	limit   int
	client  *http.Client
}

func NewAPI(marker int) *API {
	if marker == 0 {
		return nil
	}
	return &API{
		marker: marker,
	}
}

func (this *API) SetToken(token string) { this.token = token }

// Return number of remaining requests to HotelLook API. (X-Ratelimit-Remaining )
func (this *API) RequestsRemains() int { return 0 }

// Returns numeric value of API rate limit. (X-Ratelimit-Limit )
func (this *API) RequestsLimit() int { return 0 }

func (this *API) updateRemains(r *http.Response) {
	this.mu.Lock()
	this.remains, _ = strconv.Atoi(r.Header.Get("X-Ratelimit-Remaining"))
	this.limit, _ = strconv.Atoi(r.Header.Get("X-Ratelimit-Limit"))
	this.mu.Unlock()
}

// Returns urlencoded params with calculated signature.
func (this *API) withSignature(params map[string]string) string {
	var keys sort.StringSlice
	src := this.token + ":" + strconv.Itoa(this.marker)
	hash := md5.New()
	v := &url.Values{}

	if params != nil {
		for k, _ := range params {
			keys = append(keys, k)
		}
		keys.Sort()
		for _, k := range keys {
			src += ":" + params[k]
			v.Add(k, params[k])
		}
	}
	hash.Write([]byte(src))

	v.Add("marker", strconv.Itoa(this.marker))
	v.Add("signature", hex.EncodeToString(hash.Sum(nil)))
	return v.Encode()
}

// If you have no token, closed API methods will return ErrNoAccess.
func (this *API) checkAccess() error {
	if this.token == "" || this.marker == 0 {
		return ErrNoAccess
	}
	return nil
}

type LookupRequest struct {
	Query string
	// Any ISO language code (fr, de, ru...). Default is en.
	Lang string
	// city/hotel/both
	// City - cities and islands
	// Hotel - only hotels
	// Both - all values. Default.
	LookFor string
	// 10 by default.
	Limit int
	// Automatically change of keyboard map (actual for russian users). Default 1.
	ConvertCase int
}

type LookupResponse struct {
	Status  string `json:"status"`
	Results struct {
		Locations []struct {
			CityName    string   `json:"cityName"`
			FullName    string   `json:"fullName"`
			CountryCode string   `json:"countryCode,omitempty"`
			CountryName string   `json:"countryName,omitempty"`
			Iata        []string `json:"iata"`
			ID          string   `json:"id"`
			HotelsCount string   `json:"hotelsCount"`
			Location    struct {
				Lat string `json:"lat"`
				Lon string `json:"lon"`
			} `json:"location"`
			Score float64 `json:"_score,omitempty"`
		} `json:"locations"`
		Hotels []struct {
			ID           interface{} `json:"id"`
			FullName     string      `json:"fullName"`
			LocationName string      `json:"locationName"`
			Label        string      `json:"label"`
			LocationID   int         `json:"locationId"`
			Location     struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"location"`
			Score float64 `json:"_score"`
		} `json:"hotels"`
	} `json:"results"`
}

// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#31
func (this *API) Lookup(req *LookupRequest) (*LookupResponse, error) {
	const endpoint = "lookup.json?"
	v := &url.Values{}

	v.Add("query", req.Query)
	v.Add("lang", req.Lang)
	v.Add("lookFor", req.LookFor)
	if req.Limit != 0 {
		v.Add("limit", strconv.Itoa(req.Limit))
	}
	if req.ConvertCase != 0 {
		v.Add("convertCase", strconv.Itoa(req.ConvertCase))
	}
	r, err := http.Get(apiURL + endpoint + v.Encode())
	if err != nil {
		return &LookupResponse{}, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	resp := new(LookupResponse)
	if err = ffjson.NewDecoder().Decode(body, resp); err != nil {
		return &LookupResponse{}, err
	}

	return resp, nil
}

type PriceRequest struct {
	Location   string
	CheckIn    string // 2016-12-10
	CheckOut   string // 2016-12-10
	Currency   string
	LocationID int
	HotelID    int
	Hotel      string
	Adults     int // Number of adults. By default, it equals 2.
	Children   int // Childrens, age 2-18.
	Infants    int // Infants, ag 0-2.
	Limit      int
	CustomerIP net.IP
}

type PriceResponse struct {
	Stars      int     `json:"stars"`
	HotelID    int     `json:"hotelId"`
	HotelName  string  `json:"hotelName"`
	PriceAvg   float64 `json:"priceAvg"`
	PriceFrom  float64 `json:"priceFrom"`
	LocationID int     `json:"locationId"`
	Location   struct {
		Country string `json:"country"`
		State   string `json:"state"`
		Name    string `json:"name"`
		Geo     struct {
			Lon float64 `json:"lon"`
			Lat float64 `json:"lat"`
		} `json:"geo"`
	} `json:"location"`
}

// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#34
func (this *API) Price(req *PriceRequest) (*[]PriceResponse, error) {
	const endpoint = "cache.json?"

	v := &url.Values{}

	v.Add("location", req.Location)
	v.Add("checkIn", req.CheckIn)
	v.Add("checkOut", req.CheckOut)
	if req.LocationID != 0 {
		v.Add("locationId", strconv.Itoa(req.LocationID))
	}
	if req.HotelID != 0 {
		v.Add("hotleId", strconv.Itoa(req.HotelID))
	}
	if req.Hotel != "" {
		v.Add("hotel", req.Hotel)
	}
	if req.Adults != 0 {
		v.Add("adults", strconv.Itoa(req.Adults))
	}
	if req.Children != 0 {
		v.Add("children", strconv.Itoa(req.Children))
	}
	if req.Currency != "" {
		v.Add("currency", req.Currency)
	}
	if req.Infants != 0 {
		v.Add("infants", strconv.Itoa(req.Infants))
	}
	if req.Limit != 0 {
		v.Add("limit", strconv.Itoa(req.Limit))
	} else {
		req.Limit = 1
	}
	v.Add("clientIp", req.CustomerIP.String())

	r, err := http.Get(apiURL + endpoint + v.Encode())
	if err != nil {
		return nil, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	resp := make([]PriceResponse, req.Limit)
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type Countries struct {
	ID   string           `json:"id"`
	Code string           `json:"code"`
	EN   []VariationBlock `json:"EN"`
	RU   []VariationBlock `json:"RU"`
}

type VariationBlock struct {
	IsVariation string `json:"isVariation"`
	Name        string `json:"name"`
}

// Fetch contry list.
// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#41
func (this *API) Countries() (*[]Countries, error) {
	if err := this.checkAccess(); err != nil {
		return nil, err
	}
	const endpoint = "static/countries.json?"
	r, err := http.Get(apiURL + endpoint + this.withSignature(nil))
	if err != nil {
		return nil, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	resp := make([]Countries, 1)
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return nil, ErrNoAccess
	}
	return &resp, nil
}

type Cities struct {
	ID        string           `json:"id"` //locationID
	Code      string           `json:"code"`
	CountryID string           `json:"countryId"`
	Latitude  string           `json:"latitude"`
	Longitude string           `json:"longitude"`
	EN        []VariationBlock `json:"EN"`
	RU        []VariationBlock `json:"RU"`
}

// Fetch city list. Very long request.
// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#42
func (this *API) Cities() (*[]Cities, error) {
	if err := this.checkAccess(); err != nil {
		return nil, err
	}
	const endpoint = "static/locations.json?"
	r, err := http.Get(apiURL + endpoint + this.withSignature(nil))
	if err != nil {
		return nil, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	resp := make([]Cities, 2)
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return nil, ErrNoAccess
	}
	return &resp, nil
}

type Amenity struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"groupName"`
}

// Fetch available facilities.
// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#43
func (this *API) Amenities() ([]Amenity, error) {
	if err := this.checkAccess(); err != nil {
		return nil, err
	}
	const endpoint = "static/amenities.json?"
	r, err := http.Get(apiURL + endpoint + this.withSignature(nil))
	if err != nil {
		return nil, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	resp := make([]Amenity, 1)
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return nil, ErrNoAccess
	}
	return resp, nil
}

type HotelList struct {
	Timestamp float64 `json:"gen_timestamp"`
	Hotels    []Hotel `json:"hotels"`
}

type Hotel struct {
	ID            int     `json:"id"`
	CityID        int     `json:"cityId"`
	Stars         int     `json:"stars"`
	PriceFrom     int     `json:"pricefrom"`
	Rating        int     `json:"rating"`
	Popularity    int     `json:"popularity"`
	PropertyType  int     `json:"propertyType"`
	CheckIn       string  `json:"checkIn"`
	CheckOut      string  `json:"checkOut"`
	Distance      float64 `json:"distance"`
	YearOpened    int     `json:"yearOpened"`
	YearRenovated int     `json:"yearRenovated"`
	PhotoCount    int     `json:"photoCount"`
	Photos        []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"photos"`
	Facilities      []int    `json:"facilities"`
	ShortFacilities []string `json:"shortFacilities"`
	Location        struct {
		Latitude float64 `json:"lat"`
		Logitude float64 `json:"lon"`
	} `json:"location"`
	Name struct {
		EN string `json:"en"`
		RU string `json:"ru,omitempty"`
	} `json:"name"`
	CountFloors int `json:"cntFloors"`
	CountRooms  int `json:"cntRooms"`
	Address     struct {
		EN string `json:"en"`
		RU string `json:"ru,omitempty"`
	}
	Link string `json:"link"`
}

// Fetch hotel list
// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#44
func (this *API) FetchHotelList(locationId string) (*HotelList, error) {
	if err := this.checkAccess(); err != nil {
		return nil, err
	}
	v := make(map[string]string)
	v["locationId"] = locationId

	const endpoint = "static/hotels.json?"
	r, err := http.Get(apiURL + endpoint + this.withSignature(v))
	if err != nil {
		return &HotelList{}, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	resp := new(HotelList)
	if err = ffjson.NewDecoder().Decode(body, resp); err != nil {
		return &HotelList{}, ErrNoAccess
	}
	return resp, nil
}

// Fetch room types.
// Watch https://support.travelpayouts.com/hc/ru/articles/203956133-API-поиска-отелей#45
func (this *API) RoomTypes() (*interface{}, error) {
	const endpoint = "static/roomTypes.json?"
	r, err := http.Get(apiURL + endpoint + this.withSignature(nil))
	if err != nil {
		return nil, err
	}
	go this.updateRemains(r)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	resp := new(interface{})
	if err = ffjson.NewDecoder().Decode(body, resp); err != nil {
		return nil, ErrNoAccess
	}
	return resp, nil
}

func (this *API) PhotoLink(hotelId, photoId int, size string) string {
	return fmt.Sprintf("https://photo.hotellook.com/image_v2/limit/h%d_%d/%s.jpg", hotelId, photoId, size)
}

type SearchRequest struct {
	CityID        int
	HotelID       int
	IATA          string
	CheckIn       string
	CheckOut      string
	AdultsCount   int
	ChildrenCount int
	ChildAges     [3]int
	CustomerIp    string
	Currency      string
	Lang          string
	WaitForResult int
}

func (this *API) Search(req *SearchRequest) (int, error) {
	const endpoint = "search/start.json?"

	v := make(map[string]string)
	// if req.IATA == "" && (req.CityID == 0 || req.HotelID == 0) {
	// 	return "", ErrMissingParams
	// }
	v["cityId"] = strconv.Itoa(req.CityID)
	if req.HotelID != 0 {
		v["hotelId"] = strconv.Itoa(req.HotelID)
	}
	if req.WaitForResult != 0 {
		v["waitForResult"] = strconv.Itoa(req.WaitForResult)
	}
	if req.IATA != "" {
		v["iata"] = req.IATA
	}
	v["checkIn"] = req.CheckIn
	v["checkOut"] = req.CheckOut
	v["adultsCount"] = strconv.Itoa(req.AdultsCount)

	v["childrenCount"] = strconv.Itoa(req.ChildrenCount)
	if req.ChildrenCount != 0 {
		v["childAge1"] = strconv.Itoa(req.ChildAges[0])
		if req.ChildrenCount > 1 {
			v["childAge2"] = strconv.Itoa(req.ChildAges[1])
		}
		if req.ChildrenCount == 3 {
			v["childAge3"] = strconv.Itoa(req.ChildAges[2])
		}
	}
	v["lang"] = req.Lang
	v["currency"] = strings.ToUpper(req.Currency)
	v["customerIp"] = req.CustomerIp

	r, err := http.Get(apiURL + endpoint + this.withSignature(v))
	if err != nil {
		return 0, err
	}
	go this.updateRemains(r)
	var resp struct {
		SearchID int    `json:"searchId"`
		Status   string `json:"status"`
	}
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return 0, err
	}
	return resp.SearchID, nil
}

type SearchResultsRequest struct {
	SearchID int // required
	Limit    int
	Offset   int
	// Sorty by [popularity|price|name|guestScore|stars]
	SortBy string
	// If you want to sort results by descending, set it equal to -1.
	SortAsc    int
	RoomsCount int
}

type SearchResults struct {
	Status  string `json:"status"`
	Results []struct {
		FullURL          string `json:"fullUrl"`          // ссылка на отель с вашим партнерским маркером
		MaxPricePerNight int    `json:"maxPricePerNight"` // максимальная цена за ночь;
		MinPriceTotal    int    `json:"minPriceTotal"`
		MaxPrice         int    `json:"maxPrice"`
		PhotoCount       int    `json:"photoCount"`
		GuestScore       int    `json:"guestScore"`
		Address          string `json:"address"`
		ID               int    `json:"id"`
		Price            int    `json:"price"` // средняя цена за номер;
		Name             string `json:"name"`
		URL              string `json:"url"`
		Popularity       int    `json:"popularity"`
		Location         struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"location"`
		Stars    int     `json:"stars"`
		Distance float64 `json:"distance"` // расстояние от отеля до центра города;
		Rooms    []struct {
			AgencyID       string  `json:"agencyId"`
			AgencyName     string  `json:"agencyName"`
			BookingURL     string  `json:"bookingURL"`
			Type           string  `json:"type"`
			Tax            float64 `json:"tax"`
			Total          int     `json:"total"`
			Price          int     `json:"price"`
			FullBookingURL string  `json:"fullBookingURL"`
			Rating         int     `json:"rating"`
			Description    string  `json:"desc"`
			Options        struct {
				Available    int  `json:"available"`    // количество оставшихся комнат;
				Breakfast    bool `json:"breakfast"`    // включён ли завтрак;
				Refundable   bool `json":"refundable"`  // возможность возврата;
				Deposit      bool `json:"deposit"`      // оплата на сайте OTA (при бронировании);
				CardRequired bool `json:"cardRequired"` // обязательно наличие банковской карты;
				Smoking      bool `json:"smoking"`      // можно ли курить в номере;
				FreeWifi     bool `json:"freeWifi"`     // есть ли бесплатный wifi в номере;
				HotelWebsite bool `json:"hotelWebsite"` // предложение ведёт на официальный сайт отеля.
			} `json:"options"`
		} `json:"rooms"`
	} `json:"result"`
}

func (this *API) FetchSearchResults(req *SearchResultsRequest) (*SearchResults, error) {
	const endpoint = "search/getResult.json?"
	v := make(map[string]string)
	if req.SearchID == 0 {
		return nil, ErrEmptySearchID
	}
	if req.SearchID == -1 {
		body, _ := ioutil.ReadFile("./test_data.json")
		var resp SearchResults
		if err := ffjson.NewDecoder().Decode(body, &resp); err != nil {
			return &resp, err
		}
		return &resp, nil
	}
	v["searchId"] = strconv.Itoa(req.SearchID)
	if req.Limit != 0 {
		v["limit"] = strconv.Itoa(req.Limit)
	}
	if req.Offset != 0 {
		v["offset"] = strconv.Itoa(req.Offset)
	}
	if req.SortBy != "" {
		v["sortBy"] = req.SortBy
	}
	if req.SortAsc == -1 {
		v["sortAsc"] = "0"
	}
	if req.RoomsCount != 0 {
		v["roomsCount"] = strconv.Itoa(req.RoomsCount)
	}

	r, err := http.Get(apiURL + endpoint + this.withSignature(v))
	if err != nil {
		return &SearchResults{}, err
	}
	go this.updateRemains(r)

	var resp SearchResults
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err = ffjson.NewDecoder().Decode(body, &resp); err != nil {
		return &SearchResults{}, err
	}
	return &resp, nil
}
