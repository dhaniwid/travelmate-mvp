package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// AmadeusService handles communication with Amadeus Flight API
type AmadeusService struct {
	clientID     string
	clientSecret string
	baseURL      string
	tokenURL     string
	httpClient   *http.Client

	// Token caching
	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// OAuthResponse represents Amadeus OAuth token response
type OAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
	TokenType   string `json:"token_type"`
}

// FlightOfferResponse represents simplified Amadeus flight offer response
type FlightOfferResponse struct {
	Data []FlightOffer `json:"data"`
	Meta Meta          `json:"meta"`
}

type FlightOffer struct {
	Price Price `json:"price"`
}

type Price struct {
	GrandTotal string `json:"grandTotal"`
	Currency   string `json:"currency"`
}

type Meta struct {
	Count int `json:"count"`
}

// NewAmadeusService creates a new Amadeus API service
func NewAmadeusService() *AmadeusService {
	return &AmadeusService{
		clientID:     os.Getenv("AMADEUS_CLIENT_ID"),
		clientSecret: os.Getenv("AMADEUS_CLIENT_SECRET"),
		baseURL:      "https://test.api.amadeus.com", // Use test env for development
		tokenURL:     "https://test.api.amadeus.com/v1/security/oauth2/token",
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Connect handles OAuth2 authentication with token caching
func (s *AmadeusService) Connect(ctx context.Context) error {
	s.mu.RLock()
	// Check if we have a valid token
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		s.mu.RUnlock()
		return nil // Token still valid
	}
	s.mu.RUnlock()

	// Need new token
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return nil
	}

	// Validate credentials
	if s.clientID == "" || s.clientSecret == "" {
		return fmt.Errorf("AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}

	// Prepare OAuth request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", s.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Amadeus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("amadeus auth failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var oauthResp OAuthResponse
	if err := json.Unmarshal(body, &oauthResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Cache token with buffer time (subtract 60 seconds for safety)
	s.accessToken = oauthResp.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(oauthResp.ExpiresIn-60) * time.Second)

	return nil
}

// GetCheapestPrice fetches the cheapest available flight price
// Returns grandTotal as float64, currency code, and error
func (s *AmadeusService) GetCheapestPrice(ctx context.Context, origin, destination, departureDate string, returnDate *string) (float64, string, error) {
	// Ensure we have a valid token
	if err := s.Connect(ctx); err != nil {
		return 0, "", err
	}

	// Build query parameters
	params := url.Values{}
	params.Set("originLocationCode", origin)
	params.Set("destinationLocationCode", destination)
	params.Set("departureDate", departureDate) // Format: YYYY-MM-DD
	if returnDate != nil && *returnDate != "" {
		params.Set("returnDate", *returnDate)
	}
	params.Set("adults", "1")
	params.Set("currencyCode", "USD")
	params.Set("max", "1") // Only fetch cheapest option
	params.Set("nonStop", "false")

	// Build request
	endpoint := fmt.Sprintf("%s/v2/shopping/flight-offers?%s", s.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create flight search request: %w", err)
	}

	s.mu.RLock()
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	s.mu.RUnlock()

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch flight offers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read flight offers response: %w", err)
	}

	// Handle API errors
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("amadeus flight search failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var offerResp FlightOfferResponse
	if err := json.Unmarshal(body, &offerResp); err != nil {
		return 0, "", fmt.Errorf("failed to parse flight offers: %w", err)
	}

	// Validate data
	if len(offerResp.Data) == 0 {
		return 0, "", fmt.Errorf("no flight offers found for %s -> %s on %s", origin, destination, departureDate)
	}

	// Extract price
	priceStr := offerResp.Data[0].Price.GrandTotal
	currency := offerResp.Data[0].Price.Currency

	// Parse price string to float64
	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil {
		return 0, "", fmt.Errorf("invalid price format: %s", priceStr)
	}

	return price, currency, nil
}
// FlightOfferDetail represents detailed flight information
type FlightOfferDetail struct {
	ID            string  `json:"id"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	Airline       string  `json:"airline"`
	Duration      string  `json:"duration"`
	Stops         int     `json:"stops"`
	DepartureTime string  `json:"departure_time"`
	ArrivalTime   string  `json:"arrival_time"`
}

// SearchFlightOffersDetail searches for flight offers and returns detailed info
func (s *AmadeusService) SearchFlightOffersDetail(ctx context.Context, origin, destination, departureDate string, returnDate *string) ([]FlightOfferDetail, error) {
	if err := s.Connect(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("originLocationCode", origin)
	params.Set("destinationLocationCode", destination)
	params.Set("departureDate", departureDate)
	if returnDate != nil && *returnDate != "" {
		params.Set("returnDate", *returnDate)
	}
	params.Set("adults", "1")
	params.Set("currencyCode", "USD")
	params.Set("max", "5") // Fetch top 5 results
	params.Set("nonStop", "false")

	endpoint := fmt.Sprintf("%s/v2/shopping/flight-offers?%s", s.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create flight search request: %w", err)
	}

	s.mu.RLock()
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	s.mu.RUnlock()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch flight offers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read flight offers response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amadeus flight search failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Define full response structure locally to avoid cluttering package scope if only used here
	type amadeusResponse struct {
		Data []struct {
			ID string `json:"id"`
			Itineraries []struct {
				Duration string `json:"duration"`
				Segments []struct {
					Departure struct {
						At string `json:"at"`
					} `json:"departure"`
					Arrival struct {
						At string `json:"at"`
					} `json:"arrival"`
					NumberOfStops int    `json:"numberOfStops"`
					CarrierCode   string `json:"carrierCode"`
				} `json:"segments"`
			} `json:"itineraries"`
			Price struct {
				GrandTotal string `json:"grandTotal"`
				Currency   string `json:"currency"`
			} `json:"price"`
			ValidatingAirlineCodes []string `json:"validatingAirlineCodes"`
		} `json:"data"`
		Dictionaries struct {
			Carriers map[string]string `json:"carriers"`
		} `json:"dictionaries"`
	}

	var rawResp amadeusResponse
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse flight offers: %w", err)
	}

	var results []FlightOfferDetail
	for _, offer := range rawResp.Data {
		var price float64
		fmt.Sscanf(offer.Price.GrandTotal, "%f", &price)

		airlineCode := ""
		if len(offer.ValidatingAirlineCodes) > 0 {
			airlineCode = offer.ValidatingAirlineCodes[0]
		}
		airlineName := rawResp.Dictionaries.Carriers[airlineCode]
		if airlineName == "" {
			airlineName = airlineCode
		}

		// Simple logic for first itinerary (outbound)
		if len(offer.Itineraries) > 0 {
			itinerary := offer.Itineraries[0]
			stops := len(itinerary.Segments) - 1
			
			// Duration format is ISO8601 usually (PT2H30M), might need formatting on frontend or here
			// For now pass as is.
			
			detail := FlightOfferDetail{
				ID:            offer.ID,
				Price:         price,
				Currency:      offer.Price.Currency,
				Airline:       airlineName,
				Duration:      itinerary.Duration, 
				Stops:         stops,
				DepartureTime: itinerary.Segments[0].Departure.At,
				ArrivalTime:   itinerary.Segments[len(itinerary.Segments)-1].Arrival.At,
			}
			results = append(results, detail)
		}
	}

	return results, nil
}

// Location represents a simplified airport/city
type Location struct {
	Name     string `json:"name"`
	IataCode string `json:"iata_code"`
	CityName string `json:"city_name"`
	Type     string `json:"type"` // "AIRPORT" or "CITY"
}

// SearchLocations searches for cities/airports by keyword
func (s *AmadeusService) SearchLocations(ctx context.Context, keyword string) ([]Location, error) {
	if err := s.Connect(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("subType", "AIRPORT,CITY")
	params.Set("keyword", keyword)
	params.Set("page[limit]", "10")
	params.Set("view", "LIGHT")

	endpoint := fmt.Sprintf("%s/v1/reference-data/locations?%s", s.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create location search request: %w", err)
	}

	s.mu.RLock()
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	s.mu.RUnlock()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search location: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read location response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amadeus location search failed (status %d): %s", resp.StatusCode, string(body))
	}

	var locationResp struct {
		Data []struct {
			IataCode string `json:"iataCode"`
			Name     string `json:"name"`
			Address  struct {
				CityName string `json:"cityName"`
			} `json:"address"`
			SubType string `json:"subType"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &locationResp); err != nil {
		return nil, fmt.Errorf("failed to parse location response: %w", err)
	}

	var locations []Location
	for _, item := range locationResp.Data {
		locations = append(locations, Location{
			Name:     item.Name,
			IataCode: item.IataCode,
			CityName: item.Address.CityName,
			Type:     item.SubType,
		})
	}

	return locations, nil
}
