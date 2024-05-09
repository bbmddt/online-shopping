package handler

import (
	"context"
	"math/rand"

	pb "adservice/proto"
)

// Maximum number of ads to serve
const MAX_ADS_TO_SERVE = 2

// Function to be called later, returns a map of ad data
var adsMap = createAdsMap()

// Ad service struct
type AdService struct{}

// GetAds method to get ads, takes Context and request parameters, returns response and error
func (s *AdService) GetAds(context context.Context, in *pb.AdRequest) (out *pb.AdResponse, err error) {
	// All ads slice
	allAds := make([]*pb.Ad, 0)
	// Select ads based on ad categories
	if len(in.ContextKeys) > 0 {
		// Iterate over categories
		for _, category := range in.ContextKeys {
			// Get ads for the category
			ads := getAdsByCategory(category)
			// Append to ads slice
			allAds = append(allAds, ads...)
		}
		// If no ads were found, get random ads
		if len(allAds) == 0 {
			allAds = getRandomAds()
		}
	} else {
		// If no ads were found, get random ads
		allAds = getRandomAds()
	}
	// Output
	out = new(pb.AdResponse)
	// Output carries ad data
	out.Ads = allAds
	// Return
	return out, nil
}

// Get ads by category
func getAdsByCategory(category string) []*pb.Ad {
	return adsMap[category]
}

// Get random ads
func getRandomAds() []*pb.Ad {
	ads := make([]*pb.Ad, 0, MAX_ADS_TO_SERVE)
	allAds := make([]*pb.Ad, 0, 7)
	for _, ads := range adsMap {
		allAds = append(allAds, ads...)
	}
	for i := 0; i < MAX_ADS_TO_SERVE; i++ {
		ads = append(ads, allAds[rand.Intn(len(allAds))])
	}
	return ads
}

// Create ads (can also query database)
func createAdsMap() map[string][]*pb.Ad {
	hairdryer := &pb.Ad{RedirectUrl: "/product/2ZYFJ3GM2N", Text: "Hair Dryer, 50% off"}
	tankTop := &pb.Ad{RedirectUrl: "/product/66VCHSJNUP", Text: "Tank Top, 20% off"}
	candleHolder := &pb.Ad{RedirectUrl: "/product/0PUK6V6EV0", Text: "Candle Holder, 30% off"}
	bambooGlassJar := &pb.Ad{RedirectUrl: "/product/9SIQT8TOJO", Text: "Bamboo Glass Jar, 10% off"}
	watch := &pb.Ad{RedirectUrl: "/product/1YMWWN1N4O", Text: "Watch, Buy One Get One Free"}
	mug := &pb.Ad{RedirectUrl: "/product/6E92ZMYYFZ", Text: "Mug, Buy Two Get One Free"}
	loafers := &pb.Ad{RedirectUrl: "/product/L9ECAV7KIM", Text: "Loafers, Buy One Get Two Free"}
	return map[string][]*pb.Ad{
		"clothing":    {tankTop},
		"accessories": {watch},
		"footwear":    {loafers},
		"hair":        {hairdryer},
		"decor":       {candleHolder},
		"kitchen":     {bambooGlassJar, mug},
	}
}
